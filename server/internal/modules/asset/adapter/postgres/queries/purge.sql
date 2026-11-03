-- name: PurgeAssetCandidates :many
SELECT id FROM file_assets
WHERE id > sqlc.arg(after_id) AND deleted_at < sqlc.arg(cutoff)
 AND storage_metadata->'staging'='null'::jsonb AND storage_metadata->'final'='null'::jsonb AND storage_metadata->'candidates'='[]'::jsonb
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeAssets :many
WITH locked AS MATERIALIZED (
 SELECT id FROM file_assets
 WHERE id = ANY(sqlc.arg(asset_ids)::uuid[]) AND deleted_at < sqlc.arg(cutoff)
  AND storage_metadata->'staging'='null'::jsonb AND storage_metadata->'final'='null'::jsonb AND storage_metadata->'candidates'='[]'::jsonb
 ORDER BY id FOR UPDATE
)
DELETE FROM file_assets AS target
WHERE target.id IN (SELECT id FROM locked) AND target.deleted_at < sqlc.arg(cutoff)
 AND target.storage_metadata->'staging'='null'::jsonb AND target.storage_metadata->'final'='null'::jsonb AND target.storage_metadata->'candidates'='[]'::jsonb
RETURNING target.id;
