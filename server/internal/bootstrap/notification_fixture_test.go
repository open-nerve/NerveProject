package bootstrap

import (
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// seedNotification writes, directly, one notification of the work item item
// for receiver, nerve's own (an automatic archive's, M7 P1b spec 2.3), in
// the item's workspace and project and with its creator, deleted when and
// as the item is: a live item's notification is live, nerve's writing, with
// no audit account; a deleted item's is deleted at its instant by its
// deleter, deleted_at, updated_at and updated_by_id as the item's, as the
// item's deletion deletes its notifications (spec 2.2, 2.8). The deletion
// and purge guards write the notification module's rows as fixtures.
func seedNotification(t *testing.T, pool *pgxpool.Pool, item, receiver uuid.UUID) {
	t.Helper()
	if tag, err := pool.Exec(pgtest.Soon(t), `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, issue_created_by_id, sender, data,
			event_id, change_seq, updated_by_id, created_at, updated_at, deleted_at)
		SELECT $1, workspace_id, project_id, $2, id, created_by_id, 'subscribed',
			jsonb_build_object('issue', jsonb_build_object('id', id, 'name', name, 'sequence_id', sequence_id),
				'change', jsonb_build_object('field', 'archived_at', 'verb', 'updated', 'new_value', 'archive')),
			$3, 0, CASE WHEN deleted_at IS NOT NULL THEN updated_by_id END, coalesce(deleted_at, now()), coalesce(deleted_at, now()), deleted_at
		FROM issues WHERE id = $4`, uuid.NewV7(), receiver, uuid.NewV7(), item); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("seeding a notification of %s: %d rows, %v; want 1", item, tag.RowsAffected(), err)
	}
}
