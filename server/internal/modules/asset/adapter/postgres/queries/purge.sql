-- name: PurgeAssets :one
WITH candidates AS MATERIALIZED (
 SELECT candidate.id FROM file_assets AS candidate
 WHERE candidate.id>sqlc.arg(after_id) AND candidate.deleted_at<sqlc.arg(cutoff)
   AND candidate.storage_metadata->'staging'='null'::jsonb AND candidate.storage_metadata->'final'='null'::jsonb AND candidate.storage_metadata->'candidates'='[]'::jsonb
 ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
 DELETE FROM file_assets AS target WHERE target.id IN (SELECT candidates.id FROM candidates) AND target.deleted_at<sqlc.arg(cutoff)
   AND target.storage_metadata->'staging'='null'::jsonb AND target.storage_metadata->'final'='null'::jsonb AND target.storage_metadata->'candidates'='[]'::jsonb
 RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1),sqlc.arg(after_id)::uuid)::uuid AS after_id;
