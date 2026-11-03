-- name: LockImageReferenceOwners :many
SELECT id FROM projects
WHERE cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[])
ORDER BY id FOR NO KEY UPDATE;

-- name: DetachImageReferences :exec
UPDATE projects SET
cover_image_asset_id = CASE WHEN cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]) THEN NULL ELSE cover_image_asset_id END
WHERE cover_image_asset_id = ANY(sqlc.arg(asset_ids)::uuid[]);

