-- name: GetNotification :one
-- A receiver's undeleted notification, without a lock (M7 P1c spec 2.7, step 1): another receiver's, a deleted one
-- and none at all are alike not found. It answers its work item's creator too, which the guest rule decides on (step
-- 3); whether he may see it is the caller's. The primary key serves it.
SELECT id, workspace_id, project_id, issue_id, sender, triggered_by_id, data, read_at, archived_at, snoozed_till, created_at, issue_created_by_id
FROM notifications
WHERE id = sqlc.arg(id) AND receiver_id = sqlc.arg(receiver_id) AND deleted_at IS NULL;

-- name: LockNotification :one
-- The same row locked FOR NO KEY UPDATE in the transaction of its change (step 4), the row alone: a notification's
-- writes lock only its rows, a change one row (R5). The lock rereads the row's latest version, so one deleted while
-- it waited is not found. The primary key serves it.
SELECT id, workspace_id, project_id, issue_id, sender, triggered_by_id, data, read_at, archived_at, snoozed_till, created_at, issue_created_by_id
FROM notifications
WHERE id = sqlc.arg(id) AND receiver_id = sqlc.arg(receiver_id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: UpdateNotification :one
-- A receiver's change of his notification's three states (step 7), only under LockNotification's lock in the same
-- transaction and only when one changes: each written as given, at the change's moment, by the receiver as its last
-- writer. It answers the row as it now stores it; another receiver's and a deleted one are not written.
UPDATE notifications
SET read_at = sqlc.narg(read_at)::timestamptz, archived_at = sqlc.narg(archived_at)::timestamptz,
    snoozed_till = sqlc.narg(snoozed_till)::timestamptz, updated_at = sqlc.arg(at)::timestamptz, updated_by_id = sqlc.arg(receiver_id)::uuid
WHERE id = sqlc.arg(id) AND receiver_id = sqlc.arg(receiver_id)::uuid AND deleted_at IS NULL
RETURNING id, workspace_id, project_id, issue_id, sender, triggered_by_id, data, read_at, archived_at, snoozed_till, created_at;
