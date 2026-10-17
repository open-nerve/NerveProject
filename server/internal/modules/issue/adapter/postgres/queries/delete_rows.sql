-- name: DeleteProjectIssueRows :many
-- The deletion of a project's work-item side (M4 design 3.6), the work items' table: a batch of the project's
-- undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the
-- moment and by the account given (convention 5: two batches over the same rows take them in one order). The caller
-- runs the tables' statements in the global order (3.5) and, for each table, again from the batch's greatest id until
-- a batch finds none (a row deleted while the batch waits for it is passed over and the next one taken; each batch
-- reads a new snapshot). The primary key's index serves the id order and the bound. It returns the ids it deleted.
UPDATE issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectAssigneeRows :many
-- DeleteProjectIssueRows' batch, of the work items' assignees.
UPDATE issue_assignees AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_assignees r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectLabelRows :many
-- DeleteProjectIssueRows' batch, of the work items' labels.
UPDATE issue_labels AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_labels r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectSubscriberRows :many
-- DeleteProjectIssueRows' batch, of the work items' subscribers.
UPDATE issue_subscribers AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_subscribers r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectActivityRows :many
-- DeleteProjectIssueRows' batch, of the work items' activity.
UPDATE issue_activities AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_activities r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
