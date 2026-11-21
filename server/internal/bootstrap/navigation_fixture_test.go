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

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	navigationriver "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedViews writes, directly, a saved view of project and a saved view of
// its workspace, both owned by owner (M7 P3b spec 2.3), each deleted when
// and as its parent is: a live parent's view is live, the owner's writing;
// a deleted parent's is deleted at its instant by its deleter, deleted_at,
// updated_at and updated_by_id as the parent's, as the parent's deletion
// deletes its views (spec 2.7). The deletion and purge guards write the
// navigation module's rows as fixtures.
func seedViews(t *testing.T, pool *pgxpool.Pool, project, owner uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO issue_views (id, workspace_id, project_id, name, filters, display_filters, display_properties,
			owned_by_id, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
		SELECT $1::uuid, p.workspace_id, p.id, 'Urgent', '{"priority":["urgent"]}'::jsonb, $3::jsonb, $4::jsonb, $5::uuid, $5::uuid,
			CASE WHEN p.deleted_at IS NULL THEN $5::uuid ELSE p.updated_by_id END, coalesce(p.deleted_at, now()), coalesce(p.deleted_at, now()), p.deleted_at
		FROM projects AS p WHERE p.id = $6
		UNION ALL SELECT $2::uuid, w.id, NULL, 'Urgent everywhere', '{"priority":["urgent"]}'::jsonb, jsonb_set($3::jsonb, '{layout}', '"spreadsheet"'), $4::jsonb,
			$5::uuid, $5::uuid, CASE WHEN w.deleted_at IS NULL THEN $5::uuid ELSE w.updated_by_id END, coalesce(w.deleted_at, now()),
			coalesce(w.deleted_at, now()), w.deleted_at
		FROM projects AS p JOIN workspaces AS w ON w.id = p.workspace_id WHERE p.id = $6`,
		uuid.NewV7(), uuid.NewV7(), defaultDisplay, defaultProperties, owner, project); err != nil || tag.RowsAffected() != 2 {
		t.Fatalf("seeding the views of %s and its workspace: %d rows, %v; want 2", project, tag.RowsAffected(), err)
	}
}

// seedFavorites writes, directly, three favorites of owner (M7 P3c spec
// 2.3): his favorite of project, and in its workspace his folder named for
// the project and his favorite of a workspace's view in it, the favorite
// older than the folder it was moved into. Each is deleted when and as its
// parent is, as seedViews' views: the project's favorite with the project,
// the folder and the favorite in it with the workspace (spec 2.6). The
// purge guards see each table's rows deleted by its own statements as the
// runtime role; a folder's purge would take younger favorites in it
// through parent_id's CASCADE, which runs as the table's owner
// (navigation's TestPurgeUserFavorites pins that cascade).
func seedFavorites(t *testing.T, pool *pgxpool.Pool, project, owner uuid.UUID) {
	t.Helper()
	favorite, folder := uuid.NewV7(), uuid.NewV7()
	if favorite.Compare(folder) > 0 {
		favorite, folder = folder, favorite
	}
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, name,
			parent_id, sequence, created_by_id, updated_by_id, created_at, updated_at, deleted_at)
		SELECT $1::uuid, p.workspace_id, p.id, $4::uuid, 'project', p.id, NULL, NULL::uuid, 65535, $4::uuid,
			CASE WHEN p.deleted_at IS NULL THEN $4::uuid ELSE p.updated_by_id END, coalesce(p.deleted_at, now()), coalesce(p.deleted_at, now()), p.deleted_at
		FROM projects AS p WHERE p.id = $5
		UNION ALL SELECT $2::uuid, w.id, NULL, $4::uuid, 'folder', NULL, 'Pinned ' || p.identifier, NULL::uuid, 75535, $4::uuid,
			CASE WHEN w.deleted_at IS NULL THEN $4::uuid ELSE w.updated_by_id END, coalesce(w.deleted_at, now()), coalesce(w.deleted_at, now()), w.deleted_at
		FROM projects AS p JOIN workspaces AS w ON w.id = p.workspace_id WHERE p.id = $5
		UNION ALL SELECT $3::uuid, w.id, NULL, $4::uuid, 'view', gen_random_uuid(), NULL, $2::uuid, 85535, $4::uuid,
			CASE WHEN w.deleted_at IS NULL THEN $4::uuid ELSE w.updated_by_id END, coalesce(w.deleted_at, now()), coalesce(w.deleted_at, now()), w.deleted_at
		FROM projects AS p JOIN workspaces AS w ON w.id = p.workspace_id WHERE p.id = $5`,
		uuid.NewV7(), folder, favorite, owner, project); err != nil || tag.RowsAffected() != 3 {
		t.Fatalf("seeding the favorites of %s and its workspace: %d rows, %v; want 3", project, tag.RowsAffected(), err)
	}
}

// seedProjectFavorite writes, directly, owner's undeleted favorite of the
// undeleted project, as seedFavorites' first, after the project's archive
// deleted every favorite under it (M7 P3c spec 2.6, R12): the deletion
// guards need a live row under the project they delete. While the one
// seedFavorites wrote is undeleted, the partial unique index of a target's
// favorites refuses it.
func seedProjectFavorite(t *testing.T, pool *pgxpool.Pool, project, owner uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, sequence,
			created_by_id, updated_by_id, created_at, updated_at)
		SELECT $1::uuid, p.workspace_id, p.id, $2::uuid, 'project', p.id, 65535, $2::uuid, $2::uuid, now(), now()
		FROM projects AS p WHERE p.id = $3 AND p.deleted_at IS NULL`, uuid.NewV7(), owner, project); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("seeding the favorite of %s again: %d rows, %v; want 1", project, tag.RowsAffected(), err)
	}
}

// holdNavigationDeletion holds a row lock on one of the project's live
// views: the first batch of the deletion of the project's views waits for
// it, so the views stay as they were while the app and its River worker
// keep running (spec R8).
func holdNavigationDeletion(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) pgx.Tx {
	t.Helper()
	var id uuid.UUID
	if err := pool.QueryRow(t.Context(), "SELECT id FROM issue_views WHERE project_id=$1 AND deleted_at IS NULL ORDER BY id LIMIT 1", project).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return holding(t, pool, "SELECT 1 FROM issue_views WHERE id=$1 FOR UPDATE", id)
}

// waitNavigationDeletion waits for the one navigation.delete_rows job of
// the deleted parent id, a project's views or, scope "workspace", a
// workspace's own (spec 2.7): the parent's deletion moment and deleter its
// arguments, on the default queue with River's 25 attempts, completed in
// its first attempt without a failure.
func waitNavigationDeletion(t *testing.T, pool *pgxpool.Pool, scope string, id uuid.UUID) {
	t.Helper()
	parents, ok := map[string]string{"project": "projects", "workspace": "workspaces"}[scope]
	if !ok {
		t.Fatalf("no navigation deletion of the views of a %s", scope)
	}
	var at time.Time
	var by uuid.UUID
	if err := pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at,updated_by_id FROM "+parents+" WHERE id=$1", id).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	requestEventuallyWithin(t, "navigation deletion completed", 15*time.Second, func(ctx context.Context) bool {
		rows, err := pool.Query(ctx, "SELECT args,state::text,attempt,coalesce(cardinality(errors),0),queue,max_attempts FROM river_job WHERE kind=$1 AND args->>'scope'=$2 AND args->>'id'=$3",
			navigationriver.DeleteRowsKind, scope, id.String())
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
			var args navigationriver.DeleteRowsArgs
			var fields map[string]any
			if err := json.Unmarshal(raw, &args); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(raw, &fields); err != nil {
				t.Fatal(err)
			}
			if string(args.Scope) != scope || args.ID != id || !args.At.Equal(at) || args.By != by || len(fields) != 4 || queue != "default" || maxAttempts != 25 {
				t.Fatalf("navigation deletion job lost the deletion's context: %s %s/%d; want %s %s at %s by %s", raw, queue, maxAttempts, scope, id, at, by)
			}
			if failures != 0 || attempt > 1 || state == "cancelled" || state == "discarded" {
				t.Fatalf("navigation deletion job failed: %s attempt=%d failures=%d", state, attempt, failures)
			}
			count++
			complete = state == "completed"
		}
		if err := rows.Err(); err != nil {
			t.Fatal(err)
		}
		if count > 1 {
			t.Fatalf("%s %s navigation deletion jobs=%d want exactly one", scope, id, count)
		}
		return complete
	})
}

// The navigation module declares its one request kind, the deletion of the
// views, and registers its worker, with no periodic schedule; leaving the
// worker out fails the registration.
func TestNewAppNavigationJobCatalog(t *testing.T) {
	kinds := []string{"navigation.delete_rows"}
	if got := navigation.EnqueuedKinds(); !slices.Equal(got, kinds) {
		t.Fatalf("enqueued kinds=%v; want %v", got, kinds)
	}
	pool, err := postgres.NewPool(t.Context(), config.DatabaseConfig{URL: unreachableDB, MaxConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	definitions := navigation.New(navigation.Deps{Pool: pool, DeleteBatch: 1000, DeleteBudget: 45 * time.Second, RuntimeNow: time.Now,
		Logger: slog.New(slog.DiscardHandler)}).Jobs()
	if len(definitions) != 1 || definitions[0].Worker == nil || definitions[0].Periodic != nil {
		t.Fatalf("navigation jobs=%+v; want the deletion's worker alone", definitions)
	}
	cfg := jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, Logger: slog.New(slog.DiscardHandler)}
	if _, err := jobs.New(pool, cfg, definitions, kinds); err != nil {
		t.Fatalf("module request worker registration: %v", err)
	}
	if _, err := jobs.New(pool, cfg, nil, kinds); err == nil {
		t.Fatal("navigation.delete_rows declared without its worker registered")
	}
}

// navigationDeletionJobsForTest are the navigation module's request workers
// as newApp registers them, for a test that runs its own Runner in place of
// the app's: the deletion events it publishes enqueue navigation.delete_rows
// too.
func navigationDeletionJobsForTest(pool *pgxpool.Pool, cfg config.Config) []jobs.Job {
	return navigation.New(navigation.Deps{Pool: pool, JobTx: newIssueJobTx(pool, cfg), RuntimeNow: time.Now, Logger: slog.New(slog.DiscardHandler),
		DeleteBatch: cfg.Navigation.DeleteBatch, DeleteBudget: cfg.Navigation.DeleteBudget}).Jobs()
}
