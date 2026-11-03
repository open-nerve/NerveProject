-- name: BindContentAsset :one
UPDATE file_assets
SET bound_at = sqlc.arg(at),
    scope_kind = CASE WHEN scope_kind = 'new_comment' THEN 'comment' ELSE scope_kind END,
    comment_id = CASE WHEN scope_kind = 'new_comment' THEN sqlc.narg(comment_id)::uuid ELSE comment_id END,
    storage_revision = storage_revision + 1,
    updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(asset_id) AND storage_revision = sqlc.arg(revision)
  AND deleted_at IS NULL AND is_uploaded AND bound_at IS NULL
  AND ((scope_kind = 'issue' AND entity_type IN ('ISSUE_DESCRIPTION', 'ISSUE_ATTACHMENT'))
    OR (scope_kind IN ('comment', 'new_comment') AND entity_type IN ('COMMENT_DESCRIPTION', 'COMMENT_ATTACHMENT')))
  AND (scope_kind <> 'new_comment' OR sqlc.narg(comment_id)::uuid IS NOT NULL)
  AND storage_metadata->'final'->>'phase' = 'active'
RETURNING *;

-- name: DraftAssetIDs :many
SELECT id FROM file_assets
WHERE draft_issue_id = sqlc.arg(draft_id) AND deleted_at IS NULL AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: MoveDraftAsset :one
UPDATE file_assets
SET project_id = sqlc.narg(project_id)::uuid,
    scope_kind = CASE WHEN sqlc.narg(issue_id)::uuid IS NOT NULL THEN 'issue' ELSE scope_kind END,
    issue_id = sqlc.narg(issue_id)::uuid,
    draft_issue_id = CASE WHEN sqlc.narg(issue_id)::uuid IS NOT NULL THEN NULL ELSE draft_issue_id END,
    entity_type = CASE WHEN sqlc.narg(issue_id)::uuid IS NULL THEN entity_type
      WHEN entity_type = 'DRAFT_ISSUE_DESCRIPTION' THEN 'ISSUE_DESCRIPTION' ELSE 'ISSUE_ATTACHMENT' END,
    bound_at = CASE WHEN sqlc.arg(bind_asset)::boolean THEN coalesce(bound_at, sqlc.arg(at)) ELSE bound_at END,
    storage_revision = storage_revision + 1,
    updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(asset_id) AND storage_revision = sqlc.arg(revision)
  AND deleted_at IS NULL AND scope_kind = 'draft'
  AND entity_type IN ('DRAFT_ISSUE_DESCRIPTION', 'DRAFT_ISSUE_ATTACHMENT')
  AND (sqlc.narg(issue_id)::uuid IS NULL OR sqlc.narg(project_id)::uuid IS NOT NULL)
  AND ((sqlc.narg(issue_id)::uuid IS NULL AND NOT sqlc.arg(bind_asset)::boolean)
    OR (is_uploaded AND storage_metadata->'final'->>'phase' = 'active'))
RETURNING *;
