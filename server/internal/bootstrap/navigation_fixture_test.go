package bootstrap

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedViews writes, directly, a saved view of project and a saved view of
// its workspace, both owned by owner (M7 P3b spec 2.3), each deleted when
// and as its parent is: a live parent's view is live, the owner's writing;
// a deleted parent's is deleted at its instant by its deleter, deleted_at,
// updated_at and updated_by_id as the parent's, as the parent's deletion
// deletes its views (spec 2.7). The purge guards write the navigation
// module's rows as fixtures.
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
