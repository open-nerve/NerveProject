-- name: LockImageReferenceOwners :many
SELECT id FROM users
WHERE avatar_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) OR cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[])
ORDER BY id FOR NO KEY UPDATE;

-- name: DetachImageReferences :exec
UPDATE users SET
avatar_asset_id = CASE WHEN avatar_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) THEN NULL ELSE avatar_asset_id END,
cover_image_asset_id = CASE WHEN cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) THEN NULL ELSE cover_image_asset_id END
WHERE avatar_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) OR cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]);
