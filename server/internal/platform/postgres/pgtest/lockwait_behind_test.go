package pgtest_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// WaitForLockWaitBehind counts a wait for its holder's lock only: a
// connection waits for row 1 of a, which first holds, and would then wait
// for row 2, which second holds. The probe behind first returns; behind
// second it fails at its deadline, though the waiter waits on a's rows.
// Once first lets go, the waiter waits for row 2: the probe behind second
// returns, and behind first fails, the wait it saw having ended.
func TestWaitForLockWaitBehindSeesOnlyAWaitForItsHolder(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	url := pgtest.NewDatabase(t)
	if _, err := connect(t, url).Exec(ctx, "CREATE TABLE a (id int PRIMARY KEY); INSERT INTO a VALUES (1), (2)"); err != nil {
		t.Fatal(err)
	}
	first, second, waiter := connect(t, url), connect(t, url), connect(t, url)
	holding := make([]pgx.Tx, 2)
	for i, conn := range []*pgx.Conn{first, second} {
		tx, err := conn.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := tx.Exec(ctx, "SELECT id FROM a WHERE id = $1 FOR SHARE", i+1); err != nil {
			t.Fatal(err)
		}
		holding[i] = tx
	}
	waitCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	done := make(chan error, 1)
	go func() {
		tx, err := waiter.Begin(waitCtx)
		for _, id := range []int{1, 2} {
			if err == nil {
				_, err = tx.Exec(waitCtx, "SELECT id FROM a WHERE id = $1 FOR UPDATE", id)
			}
		}
		if err == nil {
			err = tx.Rollback(waitCtx)
		}
		done <- err
	}()
	// Runs before the connections close: the holders let go, and the
	// waiter gets both rows, or its context ends the wait.
	t.Cleanup(func() {
		defer cancel()
		for _, tx := range holding {
			_ = tx.Rollback(context.Background())
		}
		if err := <-done; err != nil {
			t.Errorf("the waiting connection: %v", err)
		}
	})
	pool := newPool(t, url)
	notBehind := func(holder *pgx.Conn) {
		t.Helper()
		failed := fatalOf(func(tb testing.TB) { pgtest.WaitForLockWaitBehind(tb, pool, holder.PgConn(), 300*time.Millisecond) })
		if want := fmt.Sprintf("no statement waited for a lock of backend %d within 300ms", holder.PgConn().PID()); failed != want {
			t.Errorf("WaitForLockWaitBehind(backend %d) failed with %q, want %q", holder.PgConn().PID(), failed, want)
		}
	}

	pgtest.WaitForLockWaitBehind(t, pool, first.PgConn(), 10*time.Second)
	notBehind(second)
	if err := holding[0].Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	pgtest.WaitForLockWaitBehind(t, pool, second.PgConn(), 10*time.Second)
	notBehind(first)
}
