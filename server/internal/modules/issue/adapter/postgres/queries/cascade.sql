-- name: StateInUse :one
-- Cascade.StateInUse (M4 design 3.6): whether an undeleted work item, archived or not, is in the state. deleteState
-- reads it under the project's FOR NO KEY UPDATE, which every write of the project's work items takes first (3.5), so
-- it stays as read until the deletion ends. The index on state_id serves it.
SELECT EXISTS (SELECT 1 FROM issues WHERE state_id = sqlc.arg(state_id) AND deleted_at IS NULL)::boolean AS in_use;

-- name: DetachLabels :exec
-- Cascade.DetachLabel (M4 design 3.6): the undeleted rows that put the labels on work items, soft-deleted at the moment
-- and by the account given, in one statement. deleteLabel runs it under the project's FOR NO KEY UPDATE, which every
-- write of these rows takes first (3.5): no other write reaches them meanwhile, so they are not locked by id first. The
-- index on label_id serves it.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE label_id = ANY (sqlc.arg(label_ids)::uuid[]) AND deleted_at IS NULL;
