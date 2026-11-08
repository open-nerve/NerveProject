package bootstrap

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// A start-time run of the app (waitForStartupRuns) is another backend that
// locks rows of a test's world, project rows above all, at a moment of its
// own. These stand-ins make one such run deterministic, so that a test shows
// its probe of an actor's locks or waits tells the actor from the run.

// queuedRun begins a transaction on pool that takes the row lock sql states,
// with args, and returns once that statement waits behind holder, the
// backend holding the row, as a start-time run queued there does. Nothing
// else may queue for the row first: the statement would wait behind that
// backend instead. take answers the transaction once it holds the row; the
// test's end lets it go.
func queuedRun(t *testing.T, pool *pgxpool.Pool, holder uint32, sql string, args ...any) (take func() pgx.Tx) {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	pid := tx.Conn().PgConn().PID()
	locked := make(chan error, 1)
	go func() {
		_, err := tx.Exec(context.Background(), sql, args...)
		locked <- err
	}()
	// answered is whether the statement's answer was received: until then the
	// cleanup cancels the statement and receives it itself.
	answered := false
	t.Cleanup(func() {
		if !answered {
			// Cancel the statement still waiting, not its connection, so that
			// its transaction rolls back.
			if err := tx.Conn().PgConn().CancelRequest(context.Background()); err != nil {
				t.Error(err)
			}
			<-locked
		}
		requestRelease(t, tx)
	})
	// The backends it waits behind, named, so that a failure shows who queued first.
	ctx := pgtest.Soon(t)
	for deadline := time.Now().Add(2 * time.Second); ; time.Sleep(time.Millisecond) {
		var queued bool
		var blockers string
		if err := pool.QueryRow(ctx, `SELECT $2::int = ANY (pg_blocking_pids($1::int)), coalesce((SELECT string_agg(pid || ': ' || query, '; ')
			FROM pg_stat_activity WHERE pid = ANY (pg_blocking_pids($1::int))), '')`, int64(pid), int64(holder)).Scan(&queued, &blockers); err != nil {
			t.Fatal(err)
		}
		if queued {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("stand-in run not queued behind its holder %d; it waits behind [%s]", holder, blockers)
		}
	}
	return func() pgx.Tx {
		t.Helper()
		err := receiveWithin(t, locked, 5*time.Second, "the stand-in run's row lock")
		answered = true
		if err != nil {
			t.Fatal(err)
		}
		return tx
	}
}

// letGoOnceWaitedOn rolls tx back once another backend waits for it, as a
// start-time run soon ends, and at the latest after five seconds.
func letGoOnceWaitedOn(t *testing.T, pool *pgxpool.Pool, tx pgx.Tx) {
	t.Helper()
	pid := tx.Conn().PgConn().PID()
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = tx.Rollback(context.Background()) }()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		tick := time.NewTicker(time.Millisecond)
		defer tick.Stop()
		for {
			var waited bool
			err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_stat_activity
				WHERE datname = current_database() AND $1::int = ANY (pg_blocking_pids(pid)))`, int64(pid)).Scan(&waited)
			if err != nil || waited {
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-tick.C:
			}
		}
	}()
	t.Cleanup(func() { <-done })
}
