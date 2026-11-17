package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// InsertManyTx inserts its jobs at once in ctx's transaction, each with its
// own options (M7 P1b spec R12), as InsertTx inserts one: invisible to another
// connection until the commit, gone with a rollback, and an insert's error is
// the write's.
func TestInserterInsertsManyInTheTransaction(t *testing.T) {
	for _, scenario := range []string{"commit", "rollback", "insert error"} {
		t.Run(scenario, func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			pool := openPool(t, url)
			ctx := pgtest.Soon(t)
			if _, err := pool.Exec(ctx, "CREATE TABLE inserter_probe (id int PRIMARY KEY)"); err != nil {
				t.Fatal(err)
			}
			observer, err := pgx.Connect(ctx, url)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = observer.Close(context.Background()) })
			client, err := jobs.NewInserter(pool)
			if err != nil {
				t.Fatal(err)
			}
			sentinel := errors.New("rollback probe")
			var wantErr error
			if scenario != "commit" {
				wantErr = sentinel
			}
			params := []river.InsertManyParams{
				{Args: inserterProbeArgs{Value: "first"}, InsertOpts: &river.InsertOpts{Queue: "issue_events", MaxAttempts: 7}},
				{Args: inserterProbeArgs{Value: "second"}, InsertOpts: &river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: 5}},
			}
			if scenario == "insert error" {
				params[1].Args = brokenInserterArgs{err: sentinel}
			}
			err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(ctx, func(ctx context.Context) error {
				if _, err := postgres.DB(ctx, pool).Exec(ctx, "INSERT INTO inserter_probe VALUES (1)"); err != nil {
					return err
				}
				if err := (jobInserter{client: client}).InsertManyTx(ctx, params); err != nil {
					return fmt.Errorf("publish probes: %w", err)
				}
				assertInserterRows(t, ctx, observer, 0, 0)
				var inside int
				if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&inside); err != nil {
					return err
				}
				if inside != 2 {
					t.Errorf("jobs inside business transaction = %d, want 2", inside)
				}
				if scenario == "rollback" {
					return sentinel
				}
				return nil
			})
			if !errors.Is(err, wantErr) {
				t.Fatalf("WithinTx() = %v, want error chain containing %v", err, wantErr)
			}
			if scenario != "commit" {
				assertInserterRows(t, ctx, observer, 0, 0)
				return
			}
			assertInserterRows(t, ctx, observer, 1, 2)
			rows, err := observer.Query(ctx, "SELECT kind || ' ' || state || ' ' || queue || ' ' || max_attempts || ' ' || (args->>'value') FROM river_job ORDER BY id")
			if err != nil {
				t.Fatal(err)
			}
			got, err := pgx.CollectRows(rows, pgx.RowTo[string])
			if err != nil {
				t.Fatal(err)
			}
			if want := []string{"issue_test.inserter_probe available issue_events 7 first", "issue_test.inserter_probe available default 5 second"}; !slices.Equal(got, want) {
				t.Errorf("jobs = %q; want %q", got, want)
			}
		})
	}
}

// A lost transaction fails before touching even the jobs client, as InsertTx's
// does.
func TestInserterInsertsManyOnlyInATransaction(t *testing.T) {
	err := (jobInserter{}).InsertManyTx(context.Background(), []river.InsertManyParams{{Args: inserterProbeArgs{}}})
	if !errors.Is(fmt.Errorf("publish: %w", err), errInserterRequiresTransaction) {
		t.Fatalf("InsertManyTx without transaction = %v, want recognizable transaction-required error", err)
	}
}
