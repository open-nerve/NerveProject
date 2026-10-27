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
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// draftRequestTx identifies this request's transaction, not an unrelated River
// worker. before can pause after the real precheck and before any row lock.
// The wrapped manager still owns BEGIN, context propagation and COMMIT/ROLLBACK.
type draftRequestTx struct {
	shared.TxManager
	pid    atomic.Uint32
	before *gate
}

func (d *draftRequestTx) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return d.TxManager.WithinTx(ctx, func(ctx context.Context) error {
		tx, ok := postgres.Tx(ctx)
		if !ok {
			return fmt.Errorf("request has no transaction")
		}
		d.pid.Store(tx.Conn().PgConn().PID())
		if d.before != nil {
			if err := d.before.wait(ctx); err != nil {
				return err
			}
		}
		return fn(ctx)
	})
}
func draftTx(pool *pgxpool.Pool) *draftRequestTx {
	return &draftRequestTx{TxManager: postgres.NewTxManager(pool, 2*time.Second)}
}

// A bounded condition poll is shared only by these lock proofs. It never
// substitutes elapsed time for observing a PostgreSQL or gate boundary.
func draftEventually(t *testing.T, description string, condition func(context.Context) bool) {
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
func draftPID(t *testing.T, tx *draftRequestTx) uint32 {
	t.Helper()
	draftEventually(t, "request transaction PID", func(context.Context) bool { return tx.pid.Load() != 0 })
	return tx.pid.Load()
}

// First observe a lock wait for the *known request PID*, then assert its direct
// blocker and SQL. A wrong blocker is an immediate assertion failure, not a
// timeout. Each logged edge can be composed into an actual multi-request chain.
func draftBlocked(t *testing.T, pool *pgxpool.Pool, request uint32, holder uint32, table string) {
	t.Helper()
	var blockers []int32
	var query string
	draftEventually(t, fmt.Sprintf("request PID %d lock wait", request), func(ctx context.Context) bool {
		var waits bool
		err := pool.QueryRow(ctx, `SELECT coalesce(wait_event_type='Lock',false),pg_blocking_pids(pid),query FROM pg_stat_activity WHERE pid=$1`, request).Scan(&waits, &blockers, &query)
		if err != nil {
			t.Fatal(err)
		}
		return waits
	})
	t.Logf("requestPID=%d pg_blocking_pids=%v holderPID=%d SQL=%s", request, blockers, holder, strings.Join(strings.Fields(query), " "))
	if !slices.Contains(blockers, int32(holder)) || !strings.Contains(query, table) {
		t.Fatalf("request PID %d waits behind %v on %q; want holder PID %d and %s", request, blockers, query, holder, table)
	}
}
func draftNoLock(t *testing.T, w issueWorld, table string, id uuid.UUID) {
	t.Helper()
	if pgtest.RowWaits(t, w.pool, table, id, "FOR UPDATE") {
		t.Fatalf("unexpected %s row lock", table)
	}
}
func draftExec(t *testing.T, db postgres.Querier, sql string, args ...any) {
	t.Helper()
	if _, err := db.Exec(pgtest.Soon(t), sql, args...); err != nil {
		t.Fatal(err)
	}
}
func draftCommit(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
}

func draftWrite(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange, method, path, body string) func() error {
	t.Helper()
	return w.sending(e, newIssueRoute(t, w.pool, tx), "alice", method, path, body)
}
func draftSerialize(t *testing.T, w issueWorld, first, second racingWrite, a, b *exchange, table string, after func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	g := newGate()
	ta, tb := draftTx(w.pool), draftTx(w.pool)
	fa := first(t, w, heldTx{ta, g}, a)
	fb := second(t, w, tb, b)
	_, ea, eb := serializeWaiting(t, ctx, g, fa, fb, func() { draftBlocked(t, w.pool, draftPID(t, tb), draftPID(t, ta), table) }, after)
	if ea != nil || eb != nil {
		t.Fatal(ea, eb)
	}
}
func draftPublication(id uuid.UUID) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return draftWrite(t, w, tx, e, "POST", publishPath(id), "")
	}
}
func draftMove(id, target uuid.UUID) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return draftWrite(t, w, tx, e, "PATCH", draftPath(id), `{"project_id":"`+target.String()+`"}`)
	}
}

// Release locks/barriers before collecting a failed request and before its
// pool is closed. The normal path also collects once, then disables cleanup.
func draftBackground(t *testing.T, f func() error, release func()) func() {
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
		if err := result(t, ctx, done, "draft request"); err != nil {
			t.Error(err)
		}
	}
	t.Cleanup(func() { release(); collect() })
	return collect
}
func draftOpen(g *gate) {
	select {
	case <-g.open:
	default:
		close(g.open)
	}
}

func draftRelease(t *testing.T, tx pgx.Tx) {
	t.Helper()
	if err := tx.Rollback(context.Background()); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
		t.Error(err)
	}
}
func draftWrittenTables(t *testing.T, pool *pgxpool.Pool, pid uint32) []string {
	t.Helper()
	var tables []string
	if err := pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(array_agg(DISTINCT relation::regclass::text ORDER BY relation::regclass::text),'{}') FROM pg_locks WHERE pid=$1 AND granted AND locktype='relation' AND mode='RowExclusiveLock'`, pid).Scan(&tables); err != nil {
		t.Fatal(err)
	}
	return tables
}
