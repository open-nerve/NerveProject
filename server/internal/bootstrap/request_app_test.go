package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	issuepostgres "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Only explicitly registered request IDs are observed. No SQL, parameters or
// credentials are retained by the tracer; background queries have no identity.
type requestQueries struct {
	appPool *pgxpool.Pool
	mu      sync.Mutex
	byID    map[string]*requestProbe
}

func (r *requestQueries) new() *requestProbe {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byID == nil {
		r.byID = map[string]*requestProbe{}
	}
	p := &requestProbe{id: uuid.NewV7().String(), waitTimeout: 5 * time.Second}
	r.byID[p.id] = p
	return p
}
func (r *requestQueries) lookup(id string) *requestProbe {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.byID[id]
}
func (w issueWorld) observe(req *http.Request) *requestProbe { return w.requests.observe(req) }
func (r *requestQueries) observe(req *http.Request) *requestProbe {
	p := r.new()
	req.Header.Set(httpserver.HeaderRequestID, p.id)
	return p
}
func (r *requestQueries) forRequest(req *http.Request) *requestProbe {
	return r.lookup(req.Header.Get(httpserver.HeaderRequestID))
}

func (r *requestQueries) pool(ctx context.Context, cfg config.DatabaseConfig) (*pgxpool.Pool, error) {
	source, err := postgres.NewPool(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pc := source.Config()
	source.Close()
	if pc.MinConns != 0 || pc.MinIdleConns != 0 || pc.ConnConfig.Tracer != nil {
		return nil, fmt.Errorf("observed app requires a lazy pool without an existing tracer")
	}
	// Installed before any connection can be borrowed from the returned pool.
	pc.ConnConfig.Tracer = requestQueryTracer{requests: r}
	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err == nil {
		r.appPool = pool
	}
	return pool, err
}
func startObservedApp(t *testing.T, cfg config.Config, r *requestQueries) string {
	t.Helper()
	a, err := newAppWithPoolFactory(t.Context(), cfg, slog.New(slog.DiscardHandler), migrations.FS(), testWebUI, r.pool)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(a.close)
	return runBuiltApp(t, a)
}
func newObservedIssueWorld(t *testing.T) issueWorld {
	t.Helper()
	return newObservedIssueWorldWith(t, func(*config.Config) {})
}
func newObservedIssueWorldWith(t *testing.T, configure func(*config.Config)) issueWorld {
	t.Helper()
	url := pgtest.NewDatabase(t)
	return newIssueWorldOnRequests(t, url, openPool(t, url), func(cfg *config.Config) { configure(cfg); cfg.Issue.TrimVersionsInterval = 2 * time.Second }, &requestQueries{})
}

// Select only among the 200 HTTP requests in this burst, before inspecting
// their blockers. A worker waiting on the same project cannot supply a PID.
func requestBurstBlocked(t *testing.T, pool *pgxpool.Pool, probes []*requestProbe, holder uint32) {
	t.Helper()
	var selected *requestProbe
	requestEventuallyWithin(t, "one identified burst request waiting", 5*time.Second, func(ctx context.Context) bool {
		for _, p := range probes {
			pid, seq := p.snapshot()
			if pid == 0 {
				continue
			}
			var waits bool
			if err := pool.QueryRow(ctx, `SELECT coalesce(wait_event_type='Lock',false) FROM pg_stat_activity WHERE pid=$1`, pid).Scan(&waits); err != nil {
				t.Fatal(err)
			}
			after, afterSeq := p.snapshot()
			if waits && pid == after && seq == afterSeq {
				selected = p
				return true
			}
		}
		return false
	})
	requestBlocked(t, pool, selected, holder, "projects")
}

// The live HTTP request uses the app's JWT authentication, shared pool, gate
// and workers. Its identity must not be supplied by another row waiter.
func TestFullAppRequestProbe(t *testing.T) {
	w := newObservedIssueWorld(t)
	item := w.newWorkItem(t)
	held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR UPDATE", w.web)
	jobProbe, collectJob := observedTrim(t, w.requests.appPool, func() { requestRelease(t, held) })
	jobPID := requestBlocked(t, w.pool, jobProbe, held.Conn().PgConn().PID(), "projects")

	req := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+item.String(), w.tokens["alice"], []byte(`{"name":"Observed"}`))
	probe := w.observe(req)
	ch := sendInBackground(req)
	defer requestRelease(t, held)
	requestPID := requestBlocked(t, w.pool, probe, held.Conn().PgConn().PID(), "projects")
	if requestPID == jobPID {
		t.Fatal("HTTP request inherited trim identity")
	}
	requestRelease(t, held)
	collectJob()
	a := receiveWithin(t, ch, 5*time.Second, "observed HTTP patch")
	if a.err != nil || a.res.StatusCode != 200 {
		t.Fatalf("HTTP response=%+v", a)
	}
	w.contract.CheckResponse(t, req, a.res)
	if a.res.Header.Get(httpserver.HeaderRequestID) != probe.id || probe.pid.Load() != 0 {
		t.Fatal("request ID response or query lifecycle mismatch")
	}
}

// Run the real trim worker with the raw JobTx, while exposing only its own PID.
func observedTrim(t *testing.T, pool *pgxpool.Pool, release func()) (*requestProbe, func()) {
	t.Helper()
	ports := project.Provide(pool)
	probe := &requestProbe{TxManager: newIssueJobTx(pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})}
	uc := issueapp.NewTrimVersions(issueapp.TrimVersionsDeps{Projects: issueProjects{projects: ports.IssueProjects}, Gate: issueProjects{projects: ports.IssueProjects}, Rows: issuepostgres.New(pool), Tx: probe})
	collect := requestBackground(t, func() error { return riveradapter.NewTrimVersionsWorker(uc).Work(context.Background(), nil) }, release)
	return probe, collect
}

func TestRequestPoolPreservesProductionConfiguration(t *testing.T) {
	cfg := config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 10}
	r := &requestQueries{}
	pool, err := r.pool(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	pc := pool.Config()
	if pc.MaxConns != 10 || pc.ConnConfig.RuntimeParams["jit"] != "off" || pc.ConnConfig.RuntimeParams["plan_cache_mode"] != "force_custom_plan" || pc.AfterConnect == nil || pc.ConnConfig.Tracer == nil {
		t.Fatal("observed pool lost production configuration")
	}
	conn, err := pool.Acquire(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Release()
	if _, err := conn.Exec(pgtest.Soon(t), "SET timezone='America/New_York'"); err != nil {
		t.Fatal(err)
	}
	var stamp time.Time
	if err := conn.QueryRow(pgtest.Soon(t), "SELECT '2020-01-01T00:00:00Z'::timestamptz").Scan(&stamp); err != nil || stamp.Location() != time.UTC {
		t.Fatalf("UTC codec=%v location=%v err=%v", stamp, stamp.Location(), err)
	}
}
