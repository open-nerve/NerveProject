-- name: ListDraftPreparations :many
SELECT * FROM file_assets
WHERE user_id=sqlc.arg(actor_id)::uuid AND created_by_id=sqlc.arg(actor_id)::uuid
 AND workspace_id=sqlc.arg(workspace_id)::uuid
 AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
 AND draft_issue_id=sqlc.arg(draft_id)::uuid
 AND issue_id IS NULL AND comment_id IS NULL
 AND scope_kind='draft' AND entity_type IN ('DRAFT_ISSUE_ATTACHMENT','DRAFT_ISSUE_DESCRIPTION')
 AND bound_at IS NULL AND deleted_at IS NULL
 AND id>sqlc.arg(after_id)::uuid
ORDER BY id LIMIT sqlc.arg(page_limit)::int;
