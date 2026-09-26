package pgtest_test

import (
	"context"
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// WaitForLockWait sees a lock wait in its pool's database only: every test
// has its own database, and another test's waits must not count.
func TestWaitForLockWaitSeesOnlyItsOwnDatabase(t *testing.T) {
	t.Parallel()
	waiting, idle := pgtest.NewDatabase(t), pgtest.NewDatabase(t)
	holdAndWait(t, waiting)
	idlePool := newPool(t, idle)

	pgtest.WaitForLockWait(t, newPool(t, waiting), 10*time.Second)
	failed := fatalOf(func(tb testing.TB) { pgtest.WaitForLockWait(tb, idlePool, 300*time.Millisecond) })

	if failed != "no statement waited for a lock within 300ms" {
		t.Errorf("WaitForLockWait on the other database failed with %q, want it to fail at its deadline", failed)
	}
}

// holdAndWait makes a second connection to url wait for an advisory lock
// that a first one holds, until the test ends.
func holdAndWait(t *testing.T, url string) {
	t.Helper()
	holder, waiter := connect(t, url), connect(t, url)
	if _, err := holder.Exec(context.Background(), "SELECT pg_advisory_lock(1)"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() {
		_, err := waiter.Exec(ctx, "SELECT pg_advisory_lock(1)")
		done <- err
	}()
	// Runs before the connections close: the waiter gets the lock, or its
	// context ends the wait.
	t.Cleanup(func() {
		defer cancel()
		if _, err := holder.Exec(context.Background(), "SELECT pg_advisory_unlock(1)"); err != nil {
			t.Error(err)
		}
		if err := <-done; err != nil {
			t.Errorf("the waiting connection: %v", err)
		}
	})
}

func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// fatalProbe is a testing.TB whose Fatal and Fatalf record the message and
// end the goroutine, as testing.T's do, without failing the test.
type fatalProbe struct {
	testing.TB
	message string
}

func (p *fatalProbe) Helper() {}

func (p *fatalProbe) Fatal(args ...any) {
	p.message = fmt.Sprint(args...)
	runtime.Goexit()
}

func (p *fatalProbe) Fatalf(format string, args ...any) {
	p.message = fmt.Sprintf(format, args...)
	runtime.Goexit()
}

// fatalOf runs f with a fatalProbe on a goroutine of its own and returns
// what f failed with, "" when it did not fail.
func fatalOf(f func(testing.TB)) string {
	p := &fatalProbe{}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f(p)
	}()
	<-done
	return p.message
}
