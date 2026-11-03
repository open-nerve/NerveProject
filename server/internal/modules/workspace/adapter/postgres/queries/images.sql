-- name: LockImageReferenceOwners :many
SELECT id FROM workspaces
WHERE logo_asset_id = ANY(sqlc.arg(asset_ids)::uuid[])
ORDER BY id FOR NO KEY UPDATE;

-- name: DetachImageReferences :exec
UPDATE workspaces SET
logo_asset_id = CASE WHEN logo_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) THEN NULL ELSE logo_asset_id END
WHERE logo_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]);
