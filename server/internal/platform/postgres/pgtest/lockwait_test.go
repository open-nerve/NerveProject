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

// WaitForLockWaitOn counts a wait for a row of its table only: not a wait
// for another table's row, though the waiting transaction has read that
// table; not a wait for an advisory lock; not a wait in another database,
// though a has the same OID there, as every migrated table has in every
// test database.
func TestWaitForLockWaitOnSeesOnlyWaitsForItsTablesRows(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	prepared := pgtest.NewDatabase(t)
	setup := connect(t, prepared)
	if _, err := setup.Exec(ctx, "CREATE TABLE a (id int PRIMARY KEY); CREATE TABLE b (id int PRIMARY KEY); INSERT INTO a VALUES (1)"); err != nil {
		t.Fatal(err)
	}
	if err := setup.Close(ctx); err != nil { // nobody may be connected to the database copied
		t.Fatal(err)
	}
	rows, advisory := pgtest.NewDatabaseFrom(t, prepared), pgtest.NewDatabaseFrom(t, prepared)
	holdRowAndWait(t, rows)
	holdAndWait(t, advisory)
	rowsPool, advisoryPool := newPool(t, rows), newPool(t, advisory)
	var oids [2]uint32
	for i, pool := range []*pgxpool.Pool{rowsPool, advisoryPool} {
		if err := pool.QueryRow(ctx, "SELECT 'a'::regclass::oid").Scan(&oids[i]); err != nil {
			t.Fatal(err)
		}
	}
	if oids[0] != oids[1] {
		t.Fatalf("a has the OIDs %v in the two databases, want one", oids)
	}

	pgtest.WaitForLockWaitOn(t, rowsPool, "a", 10*time.Second)
	for _, tt := range []struct {
		name  string
		pool  *pgxpool.Pool
		table string
	}{
		{"another table, which the waiter read", rowsPool, "b"},
		{"an advisory lock, in another database", advisoryPool, "a"},
	} {
		failed := fatalOf(func(tb testing.TB) { pgtest.WaitForLockWaitOn(tb, tt.pool, tt.table, 300*time.Millisecond) })
		if want := "no statement waited for a row lock of " + tt.table + " within 300ms"; failed != want {
			t.Errorf("%s: WaitForLockWaitOn failed with %q, want %q", tt.name, failed, want)
		}
	}
}

// holdRowAndWait makes a second connection to url wait for the row of a
// that a first one holds FOR NO KEY UPDATE, after reading b in the same
// transaction, until the test ends.
func holdRowAndWait(t *testing.T, url string) {
	t.Helper()
	holder, waiter := connect(t, url), connect(t, url)
	held, err := holder.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := held.Exec(context.Background(), "SELECT id FROM a WHERE id = 1 FOR NO KEY UPDATE"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	done := make(chan error, 1)
	go func() {
		tx, err := waiter.Begin(ctx)
		if err == nil {
			_, err = tx.Exec(ctx, "SELECT count(*) FROM b")
		}
		if err == nil {
			_, err = tx.Exec(ctx, "SELECT id FROM a WHERE id = 1 FOR NO KEY UPDATE")
		}
		if err == nil {
			err = tx.Rollback(ctx)
		}
		done <- err
	}()
	// Runs before the connections close: the waiter gets the row, or its
	// context ends the wait.
	t.Cleanup(func() {
		defer cancel()
		if err := held.Rollback(context.Background()); err != nil {
			t.Error(err)
		}
		if err := <-done; err != nil {
			t.Errorf("the waiting connection: %v", err)
		}
	})
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
