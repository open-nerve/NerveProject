package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// lockTimeout is the transaction's lock_timeout, as SHOW states it.
func lockTimeout(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()
	var timeout string
	if err := postgres.DB(ctx, pool).QueryRow(ctx, "SHOW lock_timeout").Scan(&timeout); err != nil {
		t.Fatal(err)
	}
	return timeout
}

// A TxManager with LockTimeout begins each transaction with lock_timeout
// set for it alone, rounded up to whole milliseconds: a statement that
// waits longer for a lock another transaction holds fails with 55P03
// (LockNotAvailable) after about that long, and its transaction rolls
// back. Its connection, the only one of its pool, is back at the default,
// no timeout, after it, and after a transaction that commits; a TxManager
// without the option leaves the default. A wait that lock_timeout does not
// end ends at the test's deadline, a failure.
func TestLockTimeoutBoundsEachWaitOfTheTransaction(t *testing.T) {
	notes := newNotes(t, 2)
	one, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: notes.Config().ConnString(), MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(one.Close)
	holder, err := notes.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = holder.Rollback(context.Background()) })
	if _, err := holder.Exec(context.Background(), "LOCK TABLE notes IN ACCESS EXCLUSIVE MODE"); err != nil {
		t.Fatal(err)
	}

	var set string
	deadline, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	start := time.Now()
	err = postgres.NewTxManager(one, commitTimeout, postgres.LockTimeout(80*time.Millisecond)).WithinTx(deadline,
		func(ctx context.Context) error {
			set = lockTimeout(t, ctx, one)
			return insertNote(ctx, one, 1)
		})
	waited := time.Since(start)
	if !postgres.LockNotAvailable(err) || set != "80ms" || waited < 80*time.Millisecond || waited > 2*time.Second {
		t.Errorf("WithinTx() = %v after %v with lock_timeout %s; want 55P03 after about 80ms, with 80ms", err, waited, set)
	}
	if after := lockTimeout(t, context.Background(), one); after != "0" {
		t.Errorf("lock_timeout after the transaction = %s, want 0", after)
	}

	for _, tt := range []struct {
		opts []postgres.Option
		want string
	}{{[]postgres.Option{postgres.LockTimeout(1500 * time.Microsecond)}, "2ms"}, {nil, "0"}} {
		var got string
		if err := postgres.NewTxManager(one, commitTimeout, tt.opts...).WithinTx(context.Background(), func(ctx context.Context) error {
			got = lockTimeout(t, ctx, one)
			return nil
		}); err != nil || got != tt.want {
			t.Errorf("WithinTx() with %d options: lock_timeout %s, %v; want %s", len(tt.opts), got, err, tt.want)
		}
	}
}

func TestLockNotAvailable(t *testing.T) {
	timedOut := &pgconn.PgError{Code: "55P03"}
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"a lock waited for too long", timedOut, true},
		{"wrapped", fmt.Errorf("update thing: %w", timedOut), true},
		{"another SQLSTATE", &pgconn.PgError{Code: "40P01"}, false},
		{"an error that is not PostgreSQL's", errors.New("55P03"), false},
		{"no error", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := postgres.LockNotAvailable(tt.err); got != tt.want {
				t.Errorf("LockNotAvailable(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
