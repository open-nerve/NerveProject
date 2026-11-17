package pgtest_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The statement probes count a backend that runs one of their statements
// only (M7 P1b spec R10). A holder holds both rows of a; ProbedElsewhere,
// another statement, whose name begins with Probed, waits behind it for
// row 1, as a River job of the app would wait for a row its test holds.
// The probes that count any backend see that wait, and so do the statement
// probes that name it; those that name Probed alone fail at their
// deadline. Once Probed waits for row 2, they return.
func TestTheStatementProbesSeeOnlyTheirStatementsWait(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	url := pgtest.NewDatabase(t)
	if _, err := connect(t, url).Exec(ctx, "CREATE TABLE a (id int PRIMARY KEY); INSERT INTO a VALUES (1), (2)"); err != nil {
		t.Fatal(err)
	}
	holder := connect(t, url)
	held, err := holder.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := held.Exec(ctx, "SELECT id FROM a FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	release := sync.OnceFunc(func() {
		if err := held.Rollback(context.Background()); err != nil {
			t.Error(err)
		}
	})
	t.Cleanup(release)
	// waits runs sql in a transaction of its own on a connection of its own.
	waits := func(sql string) {
		t.Helper()
		waiter := connect(t, url)
		waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		done := make(chan error, 1)
		go func() {
			tx, err := waiter.Begin(waitCtx)
			if err == nil {
				_, err = tx.Exec(waitCtx, sql)
			}
			if err == nil {
				err = tx.Rollback(waitCtx)
			}
			done <- err
		}()
		// Runs before the connection closes: the holder lets go, and the
		// waiter gets its row, or its context ends the wait.
		t.Cleanup(func() {
			defer cancel()
			release()
			if err := <-done; err != nil {
				t.Errorf("the waiting connection: %v", err)
			}
		})
	}
	pool := newPool(t, url)
	behind := fmt.Sprintf("a lock of backend %d", holder.PgConn().PID())
	probes := func(limit time.Duration, statements ...string) map[string]func(testing.TB) {
		return map[string]func(testing.TB){
			"a row lock of a": func(tb testing.TB) { pgtest.WaitForStatementLockWaitOn(tb, pool, "a", limit, statements...) },
			behind: func(tb testing.TB) {
				pgtest.WaitForStatementLockWaitBehind(tb, pool, holder.PgConn(), limit, statements...)
			},
		}
	}

	waits("-- name: ProbedElsewhere :exec\nSELECT id FROM a WHERE id = 1 FOR SHARE")
	pgtest.WaitForLockWaitOn(t, pool, "a", 10*time.Second)
	pgtest.WaitForLockWaitBehind(t, pool, holder.PgConn(), 10*time.Second)
	for _, probe := range probes(10*time.Second, "Probed", "ProbedElsewhere") {
		probe(t)
	}
	for wait, probe := range probes(300*time.Millisecond, "Probed") {
		if failed, want := fatalOf(t, probe), "no Probed waited for "+wait+" within 300ms"; failed != want {
			t.Errorf("the probe for Probed while ProbedElsewhere waits for %s failed with %q, want %q", wait, failed, want)
		}
	}
	waits("-- name: Probed :one\nSELECT id FROM a WHERE id = 2 FOR NO KEY UPDATE")
	for _, probe := range probes(10*time.Second, "Probed") {
		probe(t)
	}
}

// A statement probe names its waiter: one given no statement fails at
// once, as a probe that no backend could satisfy.
func TestAStatementProbeWithoutAStatementFailsAtOnce(t *testing.T) {
	t.Parallel()
	pool := newPool(t, pgtest.NewDatabase(t))
	start := time.Now()
	failed := fatalOf(t, func(tb testing.TB) { pgtest.WaitForStatementLockWaitOn(tb, pool, "pg_class", 10*time.Second) })
	if took := time.Since(start); failed != "pgtest: a statement probe names no statement" || took > 5*time.Second {
		t.Errorf("WaitForStatementLockWaitOn without a statement failed with %q after %v, want it failed at once", failed, took)
	}
}
