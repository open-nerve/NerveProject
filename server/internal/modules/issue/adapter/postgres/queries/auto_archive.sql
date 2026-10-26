-- name: AutoArchive :many
-- The caller holds the project's lock and supplies its terminal state IDs.
-- An empty state array matches nothing; no other module's table is read.
UPDATE issues
SET archived_at = sqlc.arg(at)::timestamptz,
    updated_at = sqlc.arg(at)::timestamptz, updated_by_id = NULL
WHERE project_id = sqlc.arg(project_id)
  AND deleted_at IS NULL AND archived_at IS NULL
  AND state_id = ANY (sqlc.arg(state_ids)::uuid[])
  AND updated_at <= sqlc.arg(cutoff)::timestamptz
RETURNING id;
