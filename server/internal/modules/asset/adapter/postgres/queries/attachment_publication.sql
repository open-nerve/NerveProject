-- name: BindAttachment :one
UPDATE file_assets
SET bound_at = sqlc.arg(at), storage_revision = storage_revision + 1,
    updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(asset_id) AND storage_revision = sqlc.arg(revision)
  AND deleted_at IS NULL AND is_uploaded AND bound_at IS NULL
  AND storage_metadata->'final'->>'phase' = 'active'
  AND ((scope_kind = 'issue' AND entity_type = 'ISSUE_ATTACHMENT')
    OR (scope_kind = 'comment' AND entity_type = 'COMMENT_ATTACHMENT')
    OR (scope_kind = 'draft' AND entity_type = 'DRAFT_ISSUE_ATTACHMENT'))
RETURNING *;
