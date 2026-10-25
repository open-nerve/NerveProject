package postgres_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestReadOnlySnapshotWithLockTimeout(t *testing.T) {
	for _, tt := range []struct {
		name    string
		options []postgres.Option
	}{
		{"snapshot first", []postgres.Option{postgres.ReadOnlySnapshot(), postgres.LockTimeout(1200 * time.Microsecond)}},
		{"timeout first", []postgres.Option{postgres.LockTimeout(1200 * time.Microsecond), postgres.ReadOnlySnapshot()}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pool := newNotes(t, 2)
			one, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pool.Config().ConnString(), MaxConns: 1})
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(one.Close)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			outsideCtx := ctx
			tm := postgres.NewTxManager(one, commitTimeout, tt.options...)
			var backend int
			err = tm.WithinTx(ctx, func(ctx context.Context) error {
				checkSnapshotSettings(t, ctx, one, "repeatable read", "on", "2ms")
				if err := postgres.DB(ctx, one).QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&backend); err != nil {
					return err
				}
				if n := snapshotNotes(t, ctx, one); n != 0 {
					t.Fatalf("initial count = %d, want 0", n)
				}
				// A separate connection commits after this transaction's first read.
				if err := insertNote(outsideCtx, pool, 1); err != nil {
					return err
				}
				if n := snapshotNotes(t, ctx, one); n != 0 {
					t.Errorf("snapshot after outside commit = %d, want 0", n)
				}
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			checkSnapshotSettings(t, ctx, one, "read committed", "off", "0")
			var returned int
			if err := one.QueryRow(ctx, "SELECT pg_backend_pid()").Scan(&returned); err != nil || returned != backend {
				t.Fatalf("returned backend = %d, %v; want %d", returned, err, backend)
			}
			if err := tm.WithinTx(ctx, func(ctx context.Context) error {
				if n := snapshotNotes(t, ctx, one); n != 1 {
					t.Errorf("new snapshot count = %d, want 1", n)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			// The rejected write aborts this transaction; do no further reads in it.
			err = tm.WithinTx(ctx, func(ctx context.Context) error { return insertNote(ctx, one, 2) })
			var pgErr *pgconn.PgError
			if !errors.As(err, &pgErr) || pgErr.Code != "25006" {
				t.Errorf("snapshot write = %v, want 25006 read_only_sql_transaction", err)
			}
			checkSnapshotSettings(t, ctx, one, "read committed", "off", "0")
			if n := snapshotNotes(t, ctx, one); n != 1 {
				t.Errorf("count after rejected write = %d, want 1", n)
			}
		})
	}
}

func TestReadOnlySnapshotNestedInWriteKeepsWriteTransaction(t *testing.T) {
	pool := newNotes(t, 1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	write := postgres.NewTxManager(pool, commitTimeout)
	read := postgres.NewTxManager(pool, commitTimeout, postgres.ReadOnlySnapshot(), postgres.LockTimeout(time.Second))
	err := write.WithinTx(ctx, func(ctx context.Context) error {
		return read.WithinTx(ctx, func(ctx context.Context) error {
			checkSnapshotSettings(t, ctx, pool, "read committed", "off", "0")
			return insertNote(ctx, pool, 1)
		})
	})
	if err != nil || snapshotNotes(t, ctx, pool) != 1 {
		t.Fatalf("nested write = %v, want committed insert", err)
	}
}

func snapshotNotes(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int {
	t.Helper()
	var n int
	if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM notes").Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func checkSnapshotSettings(t *testing.T, ctx context.Context, pool *pgxpool.Pool, isolation, readOnly, timeout string) {
	t.Helper()
	var gotIsolation, gotReadOnly string
	if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT current_setting('transaction_isolation'), current_setting('transaction_read_only')").Scan(&gotIsolation, &gotReadOnly); err != nil {
		t.Fatal(err)
	}
	if gotIsolation != isolation || gotReadOnly != readOnly || lockTimeout(t, ctx, pool) != timeout {
		t.Errorf("transaction settings = %s, %s, %s; want %s, %s, %s", gotIsolation, gotReadOnly, lockTimeout(t, ctx, pool), isolation, readOnly, timeout)
	}
}
