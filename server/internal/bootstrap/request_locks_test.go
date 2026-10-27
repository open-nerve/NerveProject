package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
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
	pid         atomic.Uint32
	before      *gate
	mu          sync.Mutex
	sequence    uint64
	active      uint64
	id          string
	waitTimeout time.Duration
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
	requestEventuallyWithin(t, description, 2*time.Second, condition)
}
func requestEventuallyWithin(t *testing.T, description string, timeout time.Duration, condition func(context.Context) bool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
	var query, locktype, relation, identities string
	var sequence uint64
	timeout := probe.waitTimeout
	if timeout == 0 {
		timeout = 2 * time.Second
	}
	requestEventuallyWithin(t, "identified request lock wait", timeout, func(ctx context.Context) bool {
		pid, sequence = probe.snapshot()
		if pid == 0 {
			return false
		}
		var waits bool
		err := pool.QueryRow(ctx, `SELECT coalesce(a.wait_event_type='Lock',false),pg_blocking_pids(a.pid),a.query,
   coalesce(l.locktype,''),coalesce(l.relation::regclass::text,''),
   jsonb_build_object('request_backend',a.backend_start,'request_xact',a.xact_start,
    'peers',(SELECT jsonb_agg(jsonb_build_object('pid',b.pid,'backend',b.backend_start,'xact',b.xact_start,'wait',b.wait_event_type,'sql',b.query,'blockers',pg_blocking_pids(b.pid)))
     FROM pg_stat_activity b WHERE b.pid=ANY(pg_blocking_pids(a.pid))))::text
   FROM pg_stat_activity a LEFT JOIN pg_locks l ON l.pid=a.pid AND NOT l.granted WHERE a.pid=$1`, pid).Scan(&waits, &blockers, &query, &locktype, &relation, &identities)
		if errors.Is(err, pgx.ErrNoRows) {
			return false
		}
		if err != nil {
			t.Fatal(err)
		}
		afterPID, afterSequence := probe.snapshot()
		return waits && locktype != "" && afterPID == pid && afterSequence == sequence
	})
	t.Logf("request sequence=%d identities=%s", sequence, identities)
	t.Logf("requestPID=%d pg_blocking_pids=%v holderPID=%d wait=%s relation=%s SQL=%s", pid, blockers, holder, locktype, relation, strings.Join(strings.Fields(query), " "))
	if !slices.Contains(blockers, int32(holder)) {
		chain := requestBlockerChain(t, pool, pid)
		t.Logf("requestPID=%d complete blocker chain=%v fixturePID=%d", pid, chain, holder)
		if !slices.Contains(chain, int32(holder)) {
			t.Fatalf("request PID %d blocked through %v, want fixture %d", pid, chain, holder)
		}
		after, seq := probe.snapshot()
		if after != pid || seq != sequence {
			t.Fatal("request query changed during blocker-chain inspection")
		}
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
type requestQueryTracer struct{ requests *requestQueries }
type requestQueryIdentity struct {
	probe    *requestProbe
	sequence uint64
}
type requestQueryIdentityKey struct{}

func (p *requestProbe) snapshot() (uint32, uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.pid.Load(), p.active
}
func (tr requestQueryTracer) TraceQueryStart(ctx context.Context, conn *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	p, _ := ctx.Value(requestProbeKey{}).(*requestProbe)
	if p == nil && tr.requests != nil {
		p = tr.requests.lookup(httpserver.RequestID(ctx))
	}
	if p == nil {
		return ctx
	}
	p.mu.Lock()
	p.sequence++
	p.active = p.sequence
	p.pid.Store(conn.PgConn().PID())
	identity := requestQueryIdentity{p, p.active}
	p.mu.Unlock()
	return context.WithValue(ctx, requestQueryIdentityKey{}, identity)
}
func (requestQueryTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryEndData) {
	if identity, ok := ctx.Value(requestQueryIdentityKey{}).(requestQueryIdentity); ok {
		p := identity.probe
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.active == identity.sequence {
			p.pid.Store(0)
			p.active = 0
		}
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
	publisher, err := newDispatcher(eventTypes(), issue.Subscribers(issueInserter{client: client}))
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

// A waiting worker can own the tuple queue in front of the identified request.
// This proves blocker reachability only, never ownership of a business row lock.
// Follow live transactions in this database, checking their identities and edges
// again at the end. A visited set handles cycles; a size/deadline bound rejects
// an unexpectedly large or unstable graph instead of inferring missing edges.
func requestBlockerChain(t *testing.T, pool *pgxpool.Pool, pid uint32) []int32 {
	t.Helper()
	type backend struct {
		born, started time.Time
		blockers      []int32
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	observe := func(pid int32) backend {
		var b backend
		if err := pool.QueryRow(ctx, `SELECT backend_start,xact_start,pg_blocking_pids(pid)
   FROM pg_stat_activity WHERE pid=$1 AND datname=current_database() AND xact_start IS NOT NULL`, pid).Scan(&b.born, &b.started, &b.blockers); err != nil {
			t.Fatalf("blocker PID %d no longer has a live transaction in this database: %v", pid, err)
		}
		slices.Sort(b.blockers)
		return b
	}
	seen := map[int32]backend{}
	queue := []int32{int32(pid)}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if _, ok := seen[current]; ok {
			continue
		}
		if len(seen) == 64 {
			t.Fatalf("request %d blocker graph exceeded 64 live transactions: %v", pid, seen)
		}
		b := observe(current)
		seen[current] = b
		queue = append(queue, b.blockers...)
	}
	var chain []int32
	edges := map[int32][]int32{}
	for id, b := range seen {
		after := observe(id)
		if b.born != after.born || b.started != after.started || !slices.Equal(b.blockers, after.blockers) {
			t.Fatalf("blocker PID %d changed transaction or edges during observation: before=%+v after=%+v", id, b, after)
		}
		chain = append(chain, id)
		edges[id] = b.blockers
	}
	slices.Sort(chain)
	t.Logf("requestPID=%d live same-database blocker edges=%v", pid, edges)
	return chain
}
