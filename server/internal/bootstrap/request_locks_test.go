package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// requestProbe identifies this request's transaction, not an unrelated River
// worker. before can pause after the real precheck and before any row lock.
// The wrapped manager still owns BEGIN, context propagation and COMMIT/ROLLBACK.
// Clear the identity when its callback ends, before the manager can release
// the connection; these probes observe statement locks, not commit waits.
type requestProbe struct {
	shared.TxManager
	pid    atomic.Uint32
	before *gate
}

func (d *requestProbe) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return d.TxManager.WithinTx(ctx, func(ctx context.Context) error {
		tx, ok := postgres.Tx(ctx)
		if !ok {
			return fmt.Errorf("request has no transaction")
		}
		d.pid.Store(tx.Conn().PgConn().PID())
		defer d.pid.Store(0)
		if d.before != nil {
			if err := d.before.wait(ctx); err != nil {
				return err
			}
		}
		return fn(ctx)
	})
}
func requestTx(pool *pgxpool.Pool) *requestProbe {
	return &requestProbe{TxManager: postgres.NewTxManager(pool, 2*time.Second)}
}

// A bounded condition poll is shared only by these lock proofs. It never
// substitutes elapsed time for observing a PostgreSQL or gate boundary.
func requestEventually(t *testing.T, description string, condition func(context.Context) bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	tick := time.NewTicker(time.Millisecond)
	defer tick.Stop()
	for {
		if condition(ctx) {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatalf("did not observe %s: %v", description, ctx.Err())
		case <-tick.C:
		}
	}
}
func requestPID(t *testing.T, tx *requestProbe) uint32 {
	t.Helper()
	requestEventually(t, "request transaction PID", func(context.Context) bool { return tx.pid.Load() != 0 })
	return tx.pid.Load()
}

// Identity comes only from the request's own connection. Inspect its first
// actual wait before comparing the expected blocker or table; background jobs
// may wait behind the same holder without being this request.
func requestBlocked(t *testing.T, pool *pgxpool.Pool, probe *requestProbe, holder uint32, table string) uint32 {
	t.Helper()
	return requestWait(t, pool, probe, holder, table, false)
}
func requestTableBlocked(t *testing.T, pool *pgxpool.Pool, probe *requestProbe, holder uint32, table string) uint32 {
	t.Helper()
	return requestWait(t, pool, probe, holder, table, true)
}
func requestWait(t *testing.T, pool *pgxpool.Pool, probe *requestProbe, holder uint32, table string, relationOnly bool) uint32 {
	t.Helper()
	var pid uint32
	var blockers []int32
	var query, locktype, relation string
	requestEventually(t, "identified request lock wait", func(ctx context.Context) bool {
		pid = probe.pid.Load()
		if pid == 0 {
			return false
		}
		var waits bool
		err := pool.QueryRow(ctx, `SELECT coalesce(a.wait_event_type='Lock',false),pg_blocking_pids(a.pid),a.query,
   coalesce(l.locktype,''),coalesce(l.relation::regclass::text,'')
   FROM pg_stat_activity a LEFT JOIN pg_locks l ON l.pid=a.pid AND NOT l.granted WHERE a.pid=$1`, pid).Scan(&waits, &blockers, &query, &locktype, &relation)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		return waits && locktype != "" && probe.pid.Load() == pid
	})
	t.Logf("requestPID=%d pg_blocking_pids=%v holderPID=%d wait=%s relation=%s SQL=%s", pid, blockers, holder, locktype, relation, strings.Join(strings.Fields(query), " "))
	if !slices.Contains(blockers, int32(holder)) {
		t.Fatalf("request PID %d blocked by %v, want holder %d", pid, blockers, holder)
	}
	if relationOnly || locktype == "relation" {
		if locktype != "relation" || relation != table {
			t.Fatalf("request PID %d waits for %s %s, want relation %s", pid, locktype, relation, table)
		}
	} else {
		// Row contention can wait on the owner's transaction ID rather than a
		// relation lock. The known PID, blocker and actual SQL identify this edge.
		if (locktype != "transactionid" && locktype != "tuple") || !strings.Contains(query, table) || (relation != "" && relation != table) {
			t.Fatalf("request PID %d row wait=%s %s SQL=%q, want %s", pid, locktype, relation, query, table)
		}
	}
	return pid
}

// Prechecks run before BEGIN. The context marker follows only this request;
// QueryEnd clears the active PID before the pool can reuse its connection.
type requestProbeKey struct{}
type requestQueryTracer struct{}

func (requestQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	if p, ok := ctx.Value(requestProbeKey{}).(*requestProbe); ok {
		p.pid.Store(conn.PgConn().PID())
	}
	return ctx
}
func (requestQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	if p, ok := ctx.Value(requestProbeKey{}).(*requestProbe); ok {
		p.pid.Store(0)
	}
}
func (p *requestProbe) context(ctx context.Context) context.Context {
	return context.WithValue(ctx, requestProbeKey{}, p)
}
func requestPool(t *testing.T, source *pgxpool.Pool) *pgxpool.Pool {
	t.Helper()
	cfg := source.Config()
	cfg.ConnConfig.Tracer = requestQueryTracer{}
	pool, err := pgxpool.NewWithConfig(t.Context(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// Use the real event dispatcher and River InsertTx with the request and
// transaction deadlines of the running app. Network/JWT coverage remains in
// the matrix's setup, refusal and workspace-compatibility requests.
func observedIssueRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager, cfg config.Config, gate issueapp.WriteGate) moduleRoute {
	t.Helper()
	client, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	publisher, err := newDispatcher(issue.EventTypes(), issue.Subscribers(issueInserter{client: client}))
	if err != nil {
		t.Fatal(err)
	}
	return newIssueRouteWithConfig(t, pool, tx, clock.System{}, publisher, gate, cfg)
}

// Release locks/barriers before collecting a failed request and before its
// pool is closed. The normal path also collects once, then disables cleanup.
func requestBackground(t *testing.T, f func() error, release func()) func() {
	t.Helper()
	done := run(f)
	collected := false
	collect := func() {
		if collected {
			return
		}
		collected = true
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := result(t, ctx, done, "observed request"); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { release(); collect() })
	return collect
}
func requestOpen(g *gate) {
	select {
	case <-g.open:
	default:
		close(g.open)
	}
}

func requestRelease(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Error(err)
	}
}
func requestWrittenTables(t *testing.T, pool *pgxpool.Pool, pid uint32) []string {
	t.Helper()
	var tables []string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(array_agg(DISTINCT relation::regclass::text ORDER BY relation::regclass::text),'{}') FROM pg_locks WHERE pid=$1 AND granted AND locktype='relation' AND mode='RowExclusiveLock'`, pid).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	return tables
}
