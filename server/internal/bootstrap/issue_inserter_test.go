package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type inserterProbeArgs struct {
	Value string `json:"value"`
}

func (inserterProbeArgs) Kind() string { return "issue_test.inserter_probe" }

type brokenInserterArgs struct{ err error }

func (brokenInserterArgs) Kind() string                   { return "issue_test.broken_probe" }
func (a brokenInserterArgs) MarshalJSON() ([]byte, error) { return nil, a.err }

// Opening another transaction, losing ctx's transaction, or swallowing an
// insert error must break this composition's atomic write contract.
func TestInserterCommitRollback(t *testing.T) {
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
			inserter := issueInserter{client: client}
			sentinel := errors.New("rollback probe")
			var wantErr error
			if scenario != "commit" {
				wantErr = sentinel
			}
			err = postgres.NewTxManager(pool, 2*time.Second).WithinTx(ctx, func(ctx context.Context) error {
				if _, err := postgres.DB(ctx, pool).Exec(ctx, "INSERT INTO inserter_probe VALUES (1)"); err != nil {
					return err
				}
				var args river.JobArgs = inserterProbeArgs{Value: "transactional"}
				if scenario == "insert error" {
					args = brokenInserterArgs{err: sentinel}
				}
				if err := inserter.InsertTx(ctx, args, &river.InsertOpts{Queue: "issue_events", MaxAttempts: 7}); err != nil {
					return fmt.Errorf("publish probe: %w", err)
				}
				// This connection is independent of the business transaction.
				assertInserterRows(t, ctx, observer, 0, 0)
				var inside int
				if err := postgres.DB(ctx, pool).QueryRow(ctx, "SELECT count(*) FROM river_job").Scan(&inside); err != nil {
					return err
				}
				if inside != 1 {
					t.Errorf("jobs inside business transaction = %d, want 1", inside)
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
			assertInserterRows(t, ctx, observer, 1, 1)
			var kind, state, queue, value string
			var attempts int
			if err := observer.QueryRow(ctx, "SELECT kind, state, queue, max_attempts, args->>'value' FROM river_job").Scan(&kind, &state, &queue, &attempts, &value); err != nil {
				t.Fatal(err)
			}
			if kind != "issue_test.inserter_probe" || state != "available" || queue != "issue_events" || attempts != 7 || value != "transactional" {
				t.Errorf("job = (%s, %s, %s, %d, %s), want probe available on issue_events with 7 attempts and transactional payload", kind, state, queue, attempts, value)
			}
		})
	}
}

func assertInserterRows(t *testing.T, ctx context.Context, observer *pgx.Conn, wantProbes, wantJobs int) {
	t.Helper()
	var probes, jobs int
	if err := observer.QueryRow(ctx, "SELECT (SELECT count(*) FROM inserter_probe), (SELECT count(*) FROM river_job)").Scan(&probes, &jobs); err != nil {
		t.Fatal(err)
	}
	if probes != wantProbes || jobs != wantJobs {
		t.Errorf("observer sees (%d business rows, %d jobs), want (%d, %d)", probes, jobs, wantProbes, wantJobs)
	}
}

// A lost transaction must fail before touching even the jobs client, and
// callers must recognize this programming error through wrapping.
func TestInserterRequiresTransaction(t *testing.T) {
	err := (issueInserter{}).InsertTx(context.Background(), inserterProbeArgs{}, nil)
	if !errors.Is(fmt.Errorf("publish: %w", err), errInserterRequiresTransaction) {
		t.Fatalf("InsertTx without transaction = %v, want recognizable transaction-required error", err)
	}
}
