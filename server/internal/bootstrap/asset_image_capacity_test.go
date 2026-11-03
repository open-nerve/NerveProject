package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type imageCapacityTrace struct {
	requestQueryTracer
	mu      sync.Mutex
	id      string
	queries int
	batches [][]uuid.UUID
	err     string
	timings []string
}

func (tr *imageCapacityTrace) TraceQueryStart(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	tr.mu.Lock()
	if tr.id != "" && httpserver.RequestID(ctx) == tr.id {
		tr.queries++
		name, _, _ := strings.Cut(data.SQL, "\n")
		tr.timings = append(tr.timings, fmt.Sprintf("start %s %s", time.Now().Format(time.RFC3339Nano), name))

		if strings.HasPrefix(data.SQL, "-- name: LockAssets :many") {
			if len(data.Args) != 1 {
				tr.err = "wrong asset argument count"
			} else if ids, ok := data.Args[0].([]uuid.UUID); ok {
				tr.batches = append(tr.batches, slices.Clone(ids))
			} else {
				tr.err = fmt.Sprintf("wrong asset argument type %T", data.Args[0])
			}
		}
	}
	tr.mu.Unlock()
	return tr.requestQueryTracer.TraceQueryStart(ctx, conn, data)
}

func (tr *imageCapacityTrace) TraceQueryEnd(ctx context.Context, conn *pgx.Conn, data pgx.TraceQueryEndData) {
	tr.mu.Lock()
	if tr.id != "" && httpserver.RequestID(ctx) == tr.id {
		tr.timings = append(tr.timings, fmt.Sprintf("end %s error=%v", time.Now().Format(time.RFC3339Nano), data.Err))
	}
	tr.mu.Unlock()
	tr.requestQueryTracer.TraceQueryEnd(ctx, conn, data)
}

func (tr *imageCapacityTrace) watch(id string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	tr.id, tr.queries, tr.batches, tr.err, tr.timings = id, 0, nil, "", nil
}

func (tr *imageCapacityTrace) counts() (int, [][]uuid.UUID, string) {
	tr.mu.Lock()
	defer tr.mu.Unlock()
	return tr.queries, slices.Clone(tr.batches), tr.err
}

func TestImageReadCapacity(t *testing.T) {
	requests := &requestQueries{}
	trace := &imageCapacityTrace{requestQueryTracer: requestQueryTracer{requests: requests}}
	pool := func(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
		source, err := requests.pool(ctx, cfg)
		if err != nil {
			return nil, err
		}
		pc := source.Config()
		source.Close()
		pc.ConnConfig.Tracer = trace
		p, err := pgxpool.NewWithConfig(ctx, pc)
		if err == nil {
			requests.appPool = p
		}
		return p, err
	}
	var storage *imageStorageCalls
	assets := func(ctx context.Context, c config.FilesConfig, keys *identity.Keys, logger *slog.Logger) (assetComponents, error) {
		out, err := newAssets(ctx, c, keys, logger)
		if err == nil {
			storage = &imageStorageCalls{assetStore: out.store}
			out.store = storage
		}
		return out, err
	}
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	w, _ := startImageAPIWorld(t, cfg, appFactories{pool: pool, assets: assets})
	w.requests = requests
	data := assetTestPNG(t)
	ids := make([]uuid.UUID, 0, 1001)
	create := func(n int) {
		image := uploadedAssetHTTP(t, w, "alice", "PROJECT_COVER", assetScopeBody("new_project", w.acme), "image.png", "image/png", data).Asset.ID
		w.send(t, "alice", "POST", "/api/v0/workspaces/acme/projects", fmt.Sprintf(`{"name":"Image %04d","identifier":"I%04d","cover_image_asset_id":%q}`, n, n, image), 201)
		ids = append(ids, image)
	}
	seedAt := time.Now()
	for n := range 1000 {
		create(n)
	}
	t.Logf("created 1000 real uploaded/bound API images in %s", time.Since(seedAt))
	check := func(n int) int {
		req := newRequest(t, "GET", w.base+"/api/v0/workspaces/acme/projects", w.tokens["alice"], nil)
		probe := w.observe(req)
		trace.watch(probe.id)
		beforeSigns, beforeIO := storage.signs.Load(), storage.io.Load()
		started := time.Now()
		res, err := client.Do(req)
		if err != nil {
			trace.mu.Lock()
			t.Logf("read failed: SQL=%d chunks=%d signs=%d queries=%v", trace.queries, len(trace.batches), storage.signs.Load()-beforeSigns, trace.timings)
			trace.mu.Unlock()
			t.Fatal(err)
		}
		raw, readErr := io.ReadAll(res.Body)
		closeErr := res.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatal(readErr, closeErr)
		}
		res.Body = io.NopCloser(strings.NewReader(string(raw)))
		if res.StatusCode != http.StatusOK {
			t.Fatal(res.StatusCode, string(raw))
		}
		w.contract.CheckResponse(t, req, res)
		var list struct{ Data []json.RawMessage }
		decodeAnswer(t, string(raw), &list)
		if len(list.Data) != n+1 || storage.signs.Load()-beforeSigns != int64(n) || storage.io.Load() != beforeIO {
			t.Fatal("truncated list, missing signing or storage I/O", len(list.Data), storage.signs.Load()-beforeSigns)
		}
		var gotIDs []uuid.UUID
		var expires time.Time
		for _, body := range list.Data {
			var item struct {
				Image   *uuid.UUID `json:"cover_image_asset_id"`
				URL     *string    `json:"cover_image_url"`
				Expires *time.Time `json:"cover_image_url_expires_at"`
			}
			decodeAnswer(t, string(body), &item)
			if item.Image == nil {
				if item.URL != nil || item.Expires != nil {
					t.Fatal("empty project signed")
				}
				continue
			}
			if item.URL == nil || *item.URL == "" || item.Expires == nil {
				t.Fatal("image omitted from whole collection")
			}
			if expires.IsZero() {
				expires = *item.Expires
			} else if !expires.Equal(*item.Expires) {
				t.Fatal("expiry changed inside batch")
			}
			gotIDs = append(gotIDs, *item.Image)
		}
		slices.SortFunc(gotIDs, uuid.UUID.Compare)
		wanted := slices.Clone(ids)
		slices.SortFunc(wanted, uuid.UUID.Compare)
		queries, batches, traceErr := trace.counts()
		flat := slices.Concat(batches...)
		if traceErr != "" || !slices.Equal(gotIDs, wanted) || !slices.Equal(flat, wanted) || len(batches) != (n+999)/1000 {
			t.Fatal("wrong bulk lock set", traceErr, len(gotIDs), len(flat), len(batches))
		}
		for _, batch := range batches {
			if len(batch) > 1000 {
				t.Fatal("oversized SQL chunk", len(batch))
			}
		}
		if queries > 22 {
			t.Fatal("N+1 queries", queries)
		}
		t.Logf("images=%d SQL=%d asset_chunks=%v signs=%d storage_IO=0 elapsed=%s", n, queries, []int{len(batches[0]), len(flat) - len(batches[0])}, storage.signs.Load()-beforeSigns, time.Since(started))
		return queries
	}
	firstQueries := check(1000)
	create(1000)
	if got := check(1001); got != firstQueries+1 {
		t.Fatal("adding one image must add only the second asset SQL chunk", firstQueries, got)
	}
	// Block the final chunk: no capability may be signed before it finishes.
	held := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", slices.MaxFunc(ids, uuid.UUID.Compare))
	defer requestRelease(t, held)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	req := newRequest(t, "GET", w.base+"/api/v0/workspaces/acme/projects", w.tokens["alice"], nil).WithContext(ctx)
	probe := w.observe(req)
	trace.watch(probe.id)
	before := storage.signs.Load()
	response := sendInBackground(req)
	requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "file_assets")
	_, batches, err := trace.counts()
	if err != "" || len(batches) != 2 || storage.signs.Load() != before {
		t.Fatal("signed before all batches locked", err, len(batches), storage.signs.Load()-before)
	}
	cancel()
	if got := receiveWithin(t, response, 5*time.Second, "capacity request cancellation"); got.err == nil {
		t.Fatal("cancelled bulk request succeeded")
	}
	requestRelease(t, held)
	requestEventually(t, "bulk request releases connection", func(context.Context) bool { return requests.appPool.Stat().AcquiredConns() == 0 })
	check(1001) // Also proves every acquired project gate was released.
}
