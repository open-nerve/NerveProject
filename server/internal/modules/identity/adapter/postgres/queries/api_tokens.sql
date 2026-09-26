-- name: CreateAPIToken :exec
-- An account creates its own tokens: it is their creator and last updater (M2 design 4.4).
INSERT INTO api_tokens (id, user_id, token_hash, label, description, expired_at, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(token_hash), sqlc.arg(label), sqlc.arg(description), sqlc.arg(expired_at),
        sqlc.arg(user_id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));

-- name: ListAPITokens :many
-- One page of an account's unrevoked tokens, newest first and then by id (M2 design 5.2): from the start, or after
-- the cursor's row. The row comparison casts both parameters, or sqlc types the id as a time (M2 design 3.14).
SELECT id, label, description, expired_at, last_used, created_at
FROM api_tokens
WHERE user_id = sqlc.arg(user_id) AND deleted_at IS NULL
  AND (sqlc.narg(cursor_created_at)::timestamptz IS NULL
       OR (created_at, id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(row_limit);

-- name: RevokeAPIToken :execrows
-- Revoking is a soft delete. Another account's token, or one revoked already, is not hit (M2 design 5.4).
UPDATE api_tokens
SET updated_at = sqlc.arg(now), deleted_at = sqlc.arg(now), updated_by_id = sqlc.arg(user_id)::uuid
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)::uuid AND deleted_at IS NULL;

-- name: GetAPITokenByHash :one
-- What authentication checks of a personal access token (M2 design 3.5); the use case judges it against its clock.
SELECT t.id, t.user_id, t.expired_at, t.last_used, t.deleted_at, u.is_active AS user_active
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.token_hash = sqlc.arg(token_hash);

-- name: GetAPITokenByID :one
-- The same, of the token a request authenticated with: the credential lock checks it again (M2 design 3.5).
SELECT t.id, t.user_id, t.expired_at, t.last_used, t.deleted_at, u.is_active AS user_active
FROM api_tokens t
JOIN users u ON u.id = t.user_id
WHERE t.id = sqlc.arg(id);

-- name: TouchAPIToken :exec
-- last_used, written at most once a minute (M2 design 3.5): only when it is older than stale_before. Using a token
-- changes nothing of it, so updated_at stays, as in Plane (save(update_fields=["last_used"])).
UPDATE api_tokens
SET last_used = sqlc.arg(now)::timestamptz
WHERE id = sqlc.arg(id) AND (last_used IS NULL OR last_used < sqlc.arg(stale_before)::timestamptz);
