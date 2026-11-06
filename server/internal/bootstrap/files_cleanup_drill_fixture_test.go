package bootstrap

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// filesDrill runs the production composition of one storage driver on one
// database and object store and restarts it as an operator would. Its factories
// only observe (SQL pages, the store) or, where a step says so, model elapsed
// leases and fail one object deletion; nothing in the product is switched.
type filesDrill struct {
	issueWorld
	driver  string
	cfg     config.Config
	program *app
	stop    func()
	objects assetStore
	lease   time.Duration
	fault   map[assetdomain.ObjectKey]bool
	pages   *drillPages
}

func newFilesDrill(t *testing.T, driver string) *filesDrill {
	t.Helper()
	db := pgtest.NewDatabase(t)
	d := &filesDrill{driver: driver, cfg: testConfig(t, db, false), pages: &drillPages{}}
	switch driver {
	case "s3":
		configureAssetS3(t, &d.cfg, s3test.New(t).NewBucket(t))
	case "local":
	default:
		t.Fatal("unexpected drill driver", driver)
	}
	// Pages of two make every sweep advance through several bounded batches. One
	// periodic worker runs a start's jobs in registration order: cleanup, then purge.
	d.cfg.Files.CleanupBatch, d.cfg.Cleanup.PurgeBatch, d.cfg.Jobs.PeriodicWorkers = 2, 2, 1
	observeJobStates(t, openPool(t, db))
	d.issueWorld, d.stop = startImageAPIWorld(t, d.cfg, d.factories())
	addContentMember(t, d.issueWorld)
	return d
}

// factories builds a composition with the drill's observers; lease and fault
// are read when it is built.
func (d *filesDrill) factories() appFactories {
	return appFactories{pool: d.pages.pool, assets: func(ctx context.Context, files config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
		c, err := newAssets(ctx, files, keys, logger)
		if err != nil {
			return c, err
		}
		d.objects = c.store
		if lease := d.lease; lease != 0 {
			c.runtimeNow = func() time.Time { return time.Now().Add(lease) }
		}
		if len(d.fault) != 0 {
			c.store = imageDeleteFault{assetStore: c.store, keys: d.fault}
		}
		return c, nil
	}}
}

// raw is the base and documented success status of a signed upload: the app's
// own route for local storage, the presigned absolute URL for S3.
func (d *filesDrill) raw() (string, int) {
	if d.driver == "s3" {
		return "", http.StatusOK
	}
	return d.base, http.StatusNoContent
}

// restart stops the running composition once its jobs are done and starts the
// next one after the business clock has moved by shift. Like
// restartAssetS3Cleanup it models two elapsed hours of leases on the runtime
// clock. Deleting a fault key fails in the new composition. It returns the
// cleanup and purge jobs that start ran.
func (d *filesDrill) restart(t *testing.T, shift time.Duration, fault ...assetdomain.ObjectKey) drillCycle {
	t.Helper()
	d.settled(t)
	var before int64
	if err := d.pool.QueryRow(t.Context(), "SELECT coalesce(max(id),0) FROM river_job").Scan(&before); err != nil {
		t.Fatal(err)
	}
	d.stop()
	if d.program != nil {
		d.program.close()
	}
	d.cfg.Clock.Offset += shift
	d.cfg.Server.AddrFile = filepath.Join(t.TempDir(), "addr")
	d.lease, d.fault = 2*time.Hour, map[assetdomain.ObjectKey]bool{}
	for _, key := range fault {
		d.fault[key] = true
	}
	a, err := newAppWithFactories(t.Context(), d.cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, d.factories())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	d.program = a
	d.base, d.stop = runBuiltAppWithStop(t, a)
	for _, name := range []string{"alice", "bob"} {
		var session authTokens
		decodeAnswer(t, d.send(t, "", "POST", "/api/v0/auth/login", fmt.Sprintf(`{"email":"%s@example.com","password":"Tr0ub4dor&3"}`, name), 200), &session)
		d.tokens[name] = session.AccessToken
	}
	return d.cycle(t, before)
}

// settled waits until no job is due, so a stop cannot interrupt one.
func (d *filesDrill) settled(t *testing.T) {
	t.Helper()
	requestEventuallyWithin(t, "drill jobs settled", 15*time.Second, func(ctx context.Context) bool {
		var due, failed int
		if err := d.pool.QueryRow(ctx, "SELECT count(*) FILTER (WHERE state NOT IN ('completed','cancelled','discarded')),count(*) FILTER (WHERE state IN ('cancelled','discarded')) FROM river_job").Scan(&due, &failed); err != nil {
			t.Fatal(err)
		}
		if failed != 0 {
			t.Fatalf("%d jobs were given up", failed)
		}
		return due == 0
	})
}

type drillJob struct {
	Attempt  int
	Errors   []string
	Metadata []byte
}

// drillCycle is one start's RunOnStart cleanup and purge jobs.
type drillCycle struct{ cleanup, purge drillJob }

func (d *filesDrill) cycle(t *testing.T, before int64) drillCycle {
	t.Helper()
	var jobs map[string]drillJob
	requestEventuallyWithin(t, "the start's cleanup and purge", 30*time.Second, func(ctx context.Context) bool {
		rows, err := d.pool.Query(ctx, `SELECT kind,state::text,attempt,array(SELECT e->>'error' FROM unnest(errors) e),metadata FROM river_job WHERE kind IN ('asset.cleanup','cleanup.purge_deleted') AND id>$1`, before)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		jobs = map[string]drillJob{}
		done := true
		for rows.Next() {
			var kind, state string
			var job drillJob
			if err := rows.Scan(&kind, &state, &job.Attempt, &job.Errors, &job.Metadata); err != nil {
				t.Fatal(err)
			}
			if _, twice := jobs[kind]; twice || state == "discarded" || state == "cancelled" {
				t.Fatalf("start ran %s twice or gave it up: %s", kind, state)
			}
			jobs[kind] = job
			done = done && state == "completed"
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		return done && len(jobs) == 2
	})
	return drillCycle{cleanup: jobs["asset.cleanup"], purge: jobs["cleanup.purge_deleted"]}
}

func (j drillJob) sweep(t *testing.T) assetapp.SweepResult {
	t.Helper()
	var envelope struct {
		Output assetapp.SweepResult `json:"output"`
	}
	if err := json.Unmarshal(j.Metadata, &envelope); err != nil {
		t.Fatal(err)
	}
	return envelope.Output
}

func (j drillJob) purged(t *testing.T) purgeResult {
	t.Helper()
	result, present, err := decodePurgeOutput(j.Metadata)
	if err != nil || !present {
		t.Fatalf("purge output missing: %s %v", j.Metadata, err)
	}
	return result
}

// clean requires both jobs to have finished in one attempt without errors.
func (c drillCycle) clean(t *testing.T, step string) {
	t.Helper()
	for kind, job := range map[string]drillJob{"asset.cleanup": c.cleanup, "cleanup.purge_deleted": c.purge} {
		if job.Attempt != 1 || len(job.Errors) != 0 {
			t.Fatalf("%s: %s attempt=%d errors=%q", step, kind, job.Attempt, job.Errors)
		}
	}
	if sweep := c.cleanup.sweep(t); !sweep.Complete || sweep.Failed != 0 {
		t.Fatalf("%s: cleanup %+v", step, sweep)
	}
	if purge := c.purge.purged(t); !purge.Complete {
		t.Fatalf("%s: purge %+v", step, purge)
	}
}

// slowUpload sends 3 MiB at pacedAssetBody's 160 KiB/s, about 19 s, through
// the signed raw request with a client that outlasts it, then completes the
// file and reads it back through the API.
func (d *filesDrill) slowUpload(t *testing.T, issue uuid.UUID) (uuid.UUID, string, time.Duration) {
	t.Helper()
	data := make([]byte, 3<<20)
	if _, err := rand.Read(data); err != nil {
		t.Fatal(err)
	}
	file := prepareAssetHTTP(t, d.issueWorld, "bob", "ISSUE_ATTACHMENT", assetScopeBody("issue", issue), "slow.bin", "application/octet-stream", len(data))
	base, status := d.raw()
	signed := file.uploadRequest()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	stream := &pacedAssetBody{ctx: ctx, data: data, started: make(chan time.Time, 1)}
	req, err := http.NewRequestWithContext(ctx, signed.Method, base+signed.URL, stream)
	if err != nil {
		t.Fatal(err)
	}
	req.ContentLength = int64(len(data))
	for key, value := range signed.Headers {
		req.Header.Set(key, value)
	}
	res, err := (&http.Client{Timeout: time.Minute}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	answer, err := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if err != nil || res.StatusCode != status {
		t.Fatalf("slow upload status=%d body=%q err=%v", res.StatusCode, answer, err)
	}
	took := time.Since(<-stream.started)
	budget, err := d.cfg.Files.TransferBudget(int64(len(data)))
	if err != nil || took <= 15*time.Second || d.driver == "local" && took >= budget {
		t.Fatalf("slow upload took %s, want over 15s and, on the app's own route, under its %s budget (%v)", took, budget, err)
	}
	assetAPICall(t, d.issueWorld, "bob", "POST", completeAssetPath(file.Asset.ID), `{}`, 200)
	if _, got := readAssetHTTP(t, d.issueWorld, "bob", file.Asset.ID); !bytes.Equal(got, data) {
		t.Fatal("slow upload reads back different bytes")
	}
	digest := sha256.Sum256(data)
	return file.Asset.ID, hex.EncodeToString(digest[:]), took
}

// workspaceAssets reads every file row placed in workspace.
func (d *filesDrill) workspaceAssets(t *testing.T, workspace uuid.UUID) map[uuid.UUID]assetdomain.Asset {
	t.Helper()
	out := map[uuid.UUID]assetdomain.Asset{}
	for _, id := range queryIDs(t, d.pool, "SELECT id FROM file_assets WHERE workspace_id=$1 ORDER BY id", workspace) {
		a, err := assetpg.New(d.pool).FindAsset(t.Context(), id)
		if err != nil {
			t.Fatal(err)
		}
		out[id] = a
	}
	return out
}

// expectRows compares the workspace's file rows with want. Storage revisions
// only count writes and are left out.
func (d *filesDrill) expectRows(t *testing.T, step string, workspace uuid.UUID, want map[uuid.UUID]assetdomain.Asset) {
	t.Helper()
	got := d.workspaceAssets(t, workspace)
	if len(got) != len(want) {
		t.Fatalf("%s: %d file rows, want %d", step, len(got), len(want))
	}
	for id, row := range want {
		actual, ok := got[id]
		actual.StorageRevision, row.StorageRevision = 0, 0
		if !ok || !reflect.DeepEqual(actual, row) {
			t.Fatalf("%s: file %s\n got %+v\nwant %+v", step, id, actual, row)
		}
	}
}

// expectBytes checks each key holds the bytes of its digest, or nothing ("").
func (d *filesDrill) expectBytes(t *testing.T, step string, want map[assetdomain.ObjectKey]string) {
	t.Helper()
	for key, digest := range want {
		if got := objectDigest(t, d.objects, key); got != digest {
			t.Fatalf("%s: object %s holds %q, want %q", step, key, got, digest)
		}
	}
}

func objectDigest(t *testing.T, objects assetStore, key assetdomain.ObjectKey) string {
	t.Helper()
	info, err := objects.Inspect(t.Context(), key)
	if errors.Is(err, assetdomain.ErrObjectNotFound) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	body, err := objects.Open(t.Context(), key, info.ETag)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = body.Close() }()
	sum := sha256.New()
	if _, err := io.Copy(sum, body); err != nil {
		t.Fatal(err)
	}
	return hex.EncodeToString(sum.Sum(nil))
}

// hosts reports which of the given rows still exist.
func (d *filesDrill) hosts(t *testing.T, rows map[string][]uuid.UUID) map[string]int {
	t.Helper()
	out := map[string]int{}
	for table, ids := range rows {
		var n int
		if err := d.pool.QueryRow(t.Context(), "SELECT count(*) FROM "+table+" WHERE id=ANY($1::uuid[])", ids).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[table] = n
	}
	return out
}

// drillNeighbour is workspace B with its full content tree, and alice's avatar.
type drillNeighbour struct {
	tree   contentAssetTree
	avatar uuid.UUID
}

type neighbourState struct {
	Rows, Hosts string
	Avatar      *uuid.UUID
	Bytes       map[assetdomain.ObjectKey]string
}

// state is everything of the neighbour the drill must leave alone: its file
// rows, its hosts, the avatar reference and the stored bytes.
func (d *filesDrill) state(t *testing.T, n drillNeighbour) neighbourState {
	t.Helper()
	ids := []uuid.UUID{n.avatar}
	for _, row := range n.tree.rows {
		ids = append(ids, row.ID)
	}
	s := neighbourState{Bytes: map[assetdomain.ObjectKey]string{}}
	if err := d.pool.QueryRow(t.Context(), `SELECT (SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]')::text FROM file_assets a WHERE id=ANY($1::uuid[])),
 jsonb_build_array((SELECT to_jsonb(x) FROM workspaces x WHERE id=$2),(SELECT to_jsonb(x) FROM projects x WHERE id=$3),(SELECT to_jsonb(x) FROM issues x WHERE id=$4),(SELECT to_jsonb(x) FROM issue_comments x WHERE id=$5),(SELECT to_jsonb(x) FROM draft_issues x WHERE id=$6))::text,
 (SELECT avatar_asset_id FROM users WHERE id=$7)`, ids, n.tree.workspace, n.tree.project, n.tree.issue, n.tree.comment, n.tree.draft, d.ids["alice"]).Scan(&s.Rows, &s.Hosts, &s.Avatar); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		a, err := assetpg.New(d.pool).FindAsset(t.Context(), id)
		if err != nil || a.Metadata.Final == nil || a.DeletedAt != nil || a.BoundAt == nil {
			t.Fatalf("neighbour file %s is not a live bound file: %+v %v", id, a, err)
		}
		s.Bytes[a.Metadata.Final.Key] = objectDigest(t, d.objects, a.Metadata.Final.Key)
		if s.Bytes[a.Metadata.Final.Key] != a.Metadata.Final.SHA256 {
			t.Fatalf("neighbour file %s holds other bytes", id)
		}
	}
	return s
}

// drillPages records the limit and size of every page a sweep, scope deletion
// or purge reads or deletes through the app's pool.
type drillPages struct {
	mu    sync.Mutex
	pages []drillPage
}

type drillPage struct {
	Name        string
	Limit, Rows int64
}

type drillPageKey struct{}

func (p *drillPages) pool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	source, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pc := source.Config()
	source.Close()
	pc.ConnConfig.Tracer = p
	return pgxpool.NewWithConfig(ctx, pc)
}

func (p *drillPages) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	header, _, _ := strings.Cut(data.SQL, "\n")
	fields := strings.Fields(header)
	if len(fields) < 3 || fields[0] != "--" || fields[1] != "name:" {
		return ctx
	}
	name := fields[2]
	if name != "ListRecovery" && name != "ListScope" && !strings.HasPrefix(name, "Purge") {
		return ctx
	}
	page := &drillPage{Name: name, Limit: -1}
	if n := len(data.Args); n != 0 {
		if limit, ok := data.Args[n-1].(int32); ok {
			page.Limit = int64(limit)
		}
	}
	return context.WithValue(ctx, drillPageKey{}, page)
}

func (p *drillPages) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	page, _ := ctx.Value(drillPageKey{}).(*drillPage)
	if page == nil {
		return
	}
	page.Rows = data.CommandTag.RowsAffected()
	if data.Err != nil {
		page.Rows = -1
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pages = append(p.pages, *page)
}

// bounded checks every page used its configured limit and that sweeps and
// purges each filled a page, so they advanced through several batches. It
// returns the pages and full pages per statement.
func (p *drillPages) bounded(t *testing.T, sweep, purge int) map[string][2]int {
	t.Helper()
	p.mu.Lock()
	pages := slices.Clone(p.pages)
	p.mu.Unlock()
	out := map[string][2]int{}
	full := map[bool]bool{}
	for _, page := range pages {
		sweeping := page.Name == "ListRecovery" || page.Name == "ListScope"
		limit := int64(purge)
		if sweeping {
			limit = int64(sweep)
		}
		if page.Limit != limit || page.Rows < 0 || page.Rows > limit {
			t.Fatalf("page outside its configured batch of %d: %+v", limit, page)
		}
		counts := out[page.Name]
		counts[0]++
		if page.Rows == limit {
			counts[1]++
			full[sweeping] = true
		}
		out[page.Name] = counts
	}
	if !full[true] || !full[false] {
		t.Fatalf("no full sweep or purge page: %v", out)
	}
	return out
}

// runs checks each observed run of the sweep, scope deletion and purge jobs
// ended within the configured cleanup budget, and returns their number and
// the longest.
func (d *filesDrill) runs(t *testing.T) (int, time.Duration) {
	t.Helper()
	rows, err := d.pool.Query(t.Context(), `SELECT t.job_id,t.old_state,t.new_state,t.at FROM job_observer.transitions t JOIN river_job j ON j.id=t.job_id
 WHERE j.kind IN ('asset.cleanup','asset.delete_scope','cleanup.purge_deleted') ORDER BY t.ordinal`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	started := map[int64]time.Time{}
	count, longest := 0, time.Duration(0)
	for rows.Next() {
		var id int64
		var from, to string
		var at time.Time
		if err := rows.Scan(&id, &from, &to, &at); err != nil {
			t.Fatal(err)
		}
		if to == "running" {
			started[id] = at
			continue
		}
		if from != "running" {
			continue
		}
		took := at.Sub(started[id])
		if started[id].IsZero() || took > d.cfg.Files.CleanupBudget {
			t.Fatalf("job %d ran %s, over the %s budget", id, took, d.cfg.Files.CleanupBudget)
		}
		count, longest = count+1, max(longest, took)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return count, longest
}
