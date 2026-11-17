-- name: DeleteIssueNotifications :exec
-- The notifications of a deleted work item (M7 P1b spec 2.8), in the transaction of its deletion: its undeleted rows,
-- locked FOR NO KEY UPDATE in id order before they are stamped with the deletion's moment and deleter (no deleter for
-- nerve's own deletion); a notification deleted before keeps its deletion. A fan-out holds the work item's row FOR
-- SHARE while it writes, which the deletion's UPDATE of the row waits for, so the fan-out's rows are committed and read
-- here; one that comes after reads the work item deleted and writes nothing (spec 2.9). The partial index on
-- (workspace_id, issue_id) serves it.
UPDATE notifications AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.narg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM notifications r
    WHERE r.workspace_id = sqlc.arg(workspace_id) AND r.issue_id = sqlc.arg(issue_id) AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteProjectNotifications :many
-- A batch of a deleted project's notifications (spec 2.8, R3), once none of its work items is undeleted: its undeleted
-- rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the moment and by
-- the account of the project's deletion. The caller runs it again from the batch's greatest id until a batch finds none
-- (a row deleted while the batch waits for it is passed over; each batch reads a new snapshot).
UPDATE notifications AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM notifications r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
