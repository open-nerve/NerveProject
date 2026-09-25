package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

const commitTimeout = 2 * time.Second

// newNotes returns a pool of at most maxConns connections on a new database
// holding one table, notes.
func newNotes(t *testing.T, maxConns int32) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewEmptyDatabase(t), MaxConns: maxConns})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(context.Background(), "CREATE TABLE notes (id int PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	return pool
}

func insertNote(ctx context.Context, pool *pgxpool.Pool, id int) error {
	_, err := postgres.DB(ctx, pool).Exec(ctx, "INSERT INTO notes (id) VALUES ($1)", id)
	return err
}

func countNotes(t *testing.T, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM notes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestWithinTxCommits(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		return insertNote(ctx, pool, 2)
	})

	if err != nil || countNotes(t, pool) != 2 {
		t.Errorf("WithinTx() = %v with %d notes, want nil and 2", err, countNotes(t, pool))
	}
}

func TestWithinTxRollsBackOnError(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	boom := errors.New("boom")

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		return boom
	})

	if !errors.Is(err, boom) || countNotes(t, pool) != 0 {
		t.Errorf("WithinTx() = %v with %d notes, want boom and 0", err, countNotes(t, pool))
	}
}

func TestWithinTxRollsBackOnPanic(t *testing.T) {
	pool := newNotes(t, 1)
	tm := postgres.NewTxManager(pool, commitTimeout)

	func() {
		defer func() {
			if recover() == nil {
				t.Error("the panic did not reach the caller")
			}
		}()
		_ = tm.WithinTx(context.Background(), func(ctx context.Context) error {
			if err := insertNote(ctx, pool, 1); err != nil {
				return err
			}
			panic("boom")
		})
	}()

	// One connection: the count only runs if the rollback released it.
	if n := countNotes(t, pool); n != 0 {
		t.Errorf("notes after a panic = %d, want 0", n)
	}
}

func TestNestedWithinTxJoinsTheOuterTransaction(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	boom := errors.New("boom")

	err := tm.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		inner := tm.WithinTx(ctx, func(ctx context.Context) error {
			var seen int
			// The inner call sees the outer insert: it runs in the same transaction.
			if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM notes").Scan(&seen); err != nil {
				return err
			}
			if seen != 1 {
				t.Errorf("inner transaction sees %d notes, want the outer insert", seen)
			}
			return insertNote(ctx, pool, 2)
		})
		if inner != nil {
			return inner
		}
		return boom
	})

	if !errors.Is(err, boom) || countNotes(t, pool) != 0 {
		t.Errorf("WithinTx() = %v with %d notes, want boom and 0: the outer rollback undoes the inner insert", err, countNotes(t, pool))
	}
}

func TestDBOutsideATransactionIsThePool(t *testing.T) {
	pool := newNotes(t, 1)

	if got := postgres.DB(context.Background(), pool); got != postgres.Querier(pool) {
		t.Errorf("DB() = %T, want the pool", got)
	}
}

// The request deadline cancels statements, never the COMMIT of a transaction
// whose statements have all finished (M2 design 3.6).
func TestWithinTxCommitsAfterTheContextIsCancelled(t *testing.T) {
	pool := newNotes(t, 4)
	tm := postgres.NewTxManager(pool, commitTimeout)
	ctx, cancel := context.WithCancel(context.Background())

	err := tm.WithinTx(ctx, func(ctx context.Context) error {
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		cancel() // e.g. the request deadline passes after the last statement
		return nil
	})

	if err != nil || countNotes(t, pool) != 1 {
		t.Errorf("WithinTx() = %v with %d notes, want nil and 1", err, countNotes(t, pool))
	}
}

// A failed statement, then a cancelled context: the ROLLBACK still runs, and
// the pool's only connection comes back usable.
func TestWithinTxRollsBackAfterAFailedStatementAndCancel(t *testing.T) {
	pool := newNotes(t, 1)
	tm := postgres.NewTxManager(pool, commitTimeout)
	ctx, cancel := context.WithCancel(context.Background())
	var backend int

	err := tm.WithinTx(ctx, func(ctx context.Context) error {
		if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&backend); err != nil {
			return err
		}
		if err := insertNote(ctx, pool, 1); err != nil {
			return err
		}
		err := insertNote(ctx, pool, 1) // duplicate key: the transaction is now aborted
		cancel()
		return err
	})

	if err == nil {
		t.Fatal("WithinTx() = nil, want the duplicate-key error")
	}
	var again int
	if err := pool.QueryRow(context.Background(), "SELECT pg_backend_pid()").Scan(&again); err != nil {
		t.Fatalf("the pool is not usable after the rollback: %v", err)
	}
	if again != backend {
		t.Errorf("backend %d after the rollback, want %d: the connection was not returned to the pool", again, backend)
	}
	if n := countNotes(t, pool); n != 0 {
		t.Errorf("notes = %d, want 0", n)
	}
}
