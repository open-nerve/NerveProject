-- name: LatestActivity :one
-- Read any latest live activity; an intervening field or actor interrupts description merging.
SELECT id, workspace_id, project_id, issue_id, actor_id, verb, field, old_value, new_value,
       old_identifier, new_identifier, comment, created_at, updated_at
FROM issue_activities
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: MergeDescriptionActivity :exec
-- Keep the original identity, old value and creator; advance the current content and audit.
UPDATE issue_activities
SET new_value = sqlc.arg(new_html), created_at = sqlc.arg(at)::timestamptz,
    updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
