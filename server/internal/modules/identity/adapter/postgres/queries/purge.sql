-- name: PurgeAPITokens :many
DELETE FROM api_tokens
WHERE id IN (
    SELECT candidate.id FROM api_tokens AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;
