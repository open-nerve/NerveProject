-- name: BindImage :one
UPDATE file_assets
SET bound_at = sqlc.arg(at),
    scope_kind = CASE WHEN scope_kind = 'new_project' THEN 'project' ELSE scope_kind END,
    project_id = CASE WHEN scope_kind = 'new_project' THEN sqlc.narg(project_id)::uuid ELSE project_id END,
    storage_revision = storage_revision + 1,
    updated_at = sqlc.arg(at),
    updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(asset_id)
  AND storage_revision = sqlc.arg(revision)
  AND deleted_at IS NULL AND is_uploaded AND bound_at IS NULL
  AND entity_type IN ('USER_AVATAR', 'USER_COVER', 'WORKSPACE_LOGO', 'PROJECT_COVER')
  AND storage_metadata->'final'->>'phase' = 'active'
RETURNING *;
