-- name: ShareAssetWorkspaces :many
SELECT id FROM workspaces WHERE id=ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL ORDER BY id FOR SHARE;

-- name: AssetWorkspaceFacts :many
SELECT id,deleted_at,updated_by_id AS deleted_by_id FROM workspaces WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id;
