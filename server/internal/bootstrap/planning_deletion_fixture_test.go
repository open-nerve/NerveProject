package bootstrap

import (
	"context"
	"encoding/json"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	planningriver "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedCycles writes two cycles of project by by, directly: a dated one with
// its owner, and an archived one without dates, so a deletion of the project
// reaches archived cycles too. The deletion and purge guards write the
// planning module's rows as fixtures (M6 P1 spec 2.4).
func seedCycles(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO cycles (id, workspace_id, project_id, name, start_date, end_date, owned_by_id, archived_at, created_by_id, updated_by_id)
		SELECT $1::uuid, workspace_id, id, 'Sprint', DATE '2026-11-02', DATE '2026-11-15', $2::uuid, NULL::timestamptz, $2::uuid, $2::uuid FROM projects WHERE id = $3
		UNION ALL SELECT $4::uuid, workspace_id, id, 'Done', NULL::date, NULL::date, NULL::uuid, now(), $2::uuid, $2::uuid FROM projects WHERE id = $3`,
		uuid.NewV7(), by, project, uuid.NewV7()); err != nil || tag.RowsAffected() != 2 {
		t.Fatalf("seeding the cycles of %s: %d rows, %v; want 2", project, tag.RowsAffected(), err)
	}
}

// seedModules writes a module of project by by, directly, with the rows
// under it: by as its lead and its member, and a link of it, so a deletion of
// the project reaches each of the module tables. The deletion and purge
// guards write the planning module's rows as fixtures (M6 P2 spec 2.4).
func seedModules(t *testing.T, pool *pgxpool.Pool, project, by uuid.UUID) {
	t.Helper()
	module := uuid.NewV7()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO modules (id, workspace_id, project_id, name, start_date, target_date, status, lead_id, created_by_id, updated_by_id)
			SELECT $1::uuid, workspace_id, id, 'Platform', DATE '2026-11-02', DATE '2026-11-30', 'in-progress', $2::uuid, $2::uuid, $2::uuid FROM projects WHERE id = $3`,
			[]any{module, by, project}},
		{`INSERT INTO module_members (id, workspace_id, project_id, module_id, member_id, created_by_id, updated_by_id)
			SELECT $1::uuid, workspace_id, project_id, id, $2::uuid, $2::uuid, $2::uuid FROM modules WHERE id = $3`, []any{uuid.NewV7(), by, module}},
		{`INSERT INTO module_links (id, workspace_id, project_id, module_id, title, url, created_by_id, updated_by_id)
			SELECT $1::uuid, workspace_id, project_id, id, 'Spec', 'https://nerve.dev/spec', $2::uuid, $2::uuid FROM modules WHERE id = $3`,
			[]any{uuid.NewV7(), by, module}},
	} {
		if tag, err := pool.Exec(pgtest.Soon(t), stmt.sql, stmt.args...); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("seeding the modules of %s: %s: %d rows, %v; want 1", project, stmt.sql, tag.RowsAffected(), err)
		}
	}
}

// holdPlanningDeletion holds a row lock on one of the project's live cycles:
// the planning deletion's first batch, of its first step, waits for it, so
// the planning rows stay as they were while the app and its River worker
// keep running.
func holdPlanningDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) pgx.Tx {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(t.Context(), "SELECT id FROM cycles WHERE project_id=$1 AND deleted_at IS NULL ORDER BY id LIMIT 1", project).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return holding(t, pool, "SELECT 1 FROM cycles WHERE id=$1 FOR UPDATE", id)
}

// waitPlanningDeletion waits for the one planning.delete_rows job of the
// deleted project: the project's deletion moment and deleter its arguments,
// on the default queue with River's 25 attempts, completed in its first
// attempt without a failure.
func waitPlanningDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) {
	t.Helper()
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_by_id FROM projects WHERE id=$1", project).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	requestEventuallyWithin(t, "planning deletion completed", 15*time.Second, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, "SELECT args,state::text,attempt,coalesce(cardinality(errors),0),queue,max_attempts FROM river_job WHERE kind=$1 AND args->>'project_id'=$2",
			planningriver.DeleteRowsKind, project.String())
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		count, complete := 0, false
		for rows.Next() {
			var raw []byte
			var state, queue string
			var attempt, failures, maxAttempts int
			if err := rows.Scan(&raw, &state, &attempt, &failures, &queue, &maxAttempts); err != nil {
				t.Fatal(err)
			}
			var args planningriver.DeleteRowsArgs
			var fields map[string]any
			if err := json.Unmarshal(raw, &args); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if args.ProjectID != project || !args.At.Equal(at) || args.By != by || len(fields) != 3 || queue != "default" || maxAttempts != 25 {
				t.Fatalf("planning deletion job lost the deletion's context: %s %s/%d; want project %s at %s by %s", raw, queue, maxAttempts, project, at, by)
			}
			if failures != 0 || attempt > 1 || state == "cancelled" || state == "discarded" {
				t.Fatalf("planning deletion job failed: %s attempt=%d failures=%d", state, attempt, failures)
			}
			count++
			complete = state == "completed"
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if count > 1 {
			t.Fatalf("project %s planning deletion jobs=%d want exactly one", project, count)
		}
		return complete
	})
}

// The planning module declares its request kind and registers its worker,
// with no periodic schedule; leaving the worker out fails the registration.
func TestNewAppPlanningJobCatalog(t *testing.T) {
	kinds := []string{"planning.crawl_link", "planning.delete_rows"}
	if got := planning.EnqueuedKinds(); !slices.Equal(got, kinds) {
		t.Fatalf("enqueued kinds=%v; want %v", got, kinds)
	}
	pool, err := postgres.NewPool(t.Context(), config.DatabaseConfig{URL: unreachableDB, MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	module := planning.New(planning.Deps{Pool: pool, DeleteBatch: 1000, DeleteBudget: 45 * time.Second, RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler)})
	definitions := module.Jobs()
	if len(definitions) != 2 || slices.ContainsFunc(definitions, func(j jobs.Job) bool { return j.Worker == nil || j.Periodic != nil }) {
		t.Fatalf("planning jobs=%+v; want the crawl and the deletion workers alone", definitions)
	}
	cfg := jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, Logger: slog.New(slog.DiscardHandler)}
	if _, err := jobs.New(pool, cfg, definitions, kinds); err != nil {
		t.Fatalf("module request worker registration: %v", err)
	}
	for _, kind := range kinds {
		if _, err := jobs.New(pool, cfg, nil, []string{kind}); err == nil {
			t.Fatalf("%s declared without its worker registered", kind)
		}
	}
}

// planningDeletionJobsForTest are the planning module's request workers as
// newApp registers them, for a test that runs its own Runner in place of the
// app's: the deletion events it publishes enqueue planning.delete_rows too.
func planningDeletionJobsForTest(pool *pgxpool.Pool, cfg config.Config) []jobs.Job {
	return planning.New(planning.Deps{Pool: pool, JobTx: newIssueJobTx(pool, cfg), RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler),
		DeleteBatch: cfg.Planning.DeleteBatch, DeleteBudget: cfg.Planning.DeleteBudget}).Jobs()
}
