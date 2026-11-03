-- name: ListAttachments :many
SELECT id, attributes, size, created_at, created_by_id FROM file_assets
WHERE workspace_id=sqlc.arg(workspace_id)::uuid
 AND project_id IS NOT DISTINCT FROM sqlc.narg(project_id)::uuid
 AND issue_id IS NOT DISTINCT FROM sqlc.narg(issue_id)::uuid
 AND comment_id IS NOT DISTINCT FROM sqlc.narg(comment_id)::uuid
 AND draft_issue_id IS NOT DISTINCT FROM sqlc.narg(draft_id)::uuid
 AND scope_kind=sqlc.arg(scope_kind)::text AND entity_type=sqlc.arg(purpose)::text
 AND is_uploaded=true AND bound_at IS NOT NULL AND deleted_at IS NULL
 AND id>sqlc.arg(after_id)::uuid
ORDER BY id LIMIT sqlc.arg(page_limit)::int;

-- name: AttachmentCounts :many
SELECT issue_id, count(*)::bigint AS attachment_count FROM file_assets
WHERE issue_id=ANY(sqlc.arg(issue_ids)::uuid[])
 AND scope_kind='issue' AND entity_type='ISSUE_ATTACHMENT'
 AND is_uploaded=true AND bound_at IS NOT NULL AND deleted_at IS NULL
GROUP BY issue_id ORDER BY issue_id;

-- name: ProjectAttachmentCounts :many
SELECT issue_id, count(*)::bigint AS attachment_count FROM file_assets
WHERE project_id=ANY(sqlc.arg(project_ids)::uuid[]) AND issue_id>sqlc.arg(after_id)::uuid
 AND scope_kind='issue' AND entity_type='ISSUE_ATTACHMENT'
 AND is_uploaded=true AND bound_at IS NOT NULL AND deleted_at IS NULL
GROUP BY issue_id ORDER BY issue_id LIMIT sqlc.arg(page_limit)::int;
