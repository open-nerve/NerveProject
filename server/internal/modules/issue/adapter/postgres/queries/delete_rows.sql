-- name: DeleteProjectIssueRows :many
-- The deletion of a project's work-item side (M4 design 3.6), the work items' table: a batch of the project's
-- undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the
-- moment and by the account given (convention 5: two batches over the same rows take them in one order). The caller
-- runs it again from the batch's greatest id until a batch finds none (a row deleted while the batch waits for it is
-- passed over and the next one taken; each batch reads a new snapshot). The primary key's index serves the id order
-- and the bound. It returns the ids it deleted.
UPDATE issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
