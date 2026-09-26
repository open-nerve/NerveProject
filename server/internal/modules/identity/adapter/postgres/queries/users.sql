-- name: CreateUser :exec
-- The other columns take their defaults; the audit columns come from the use case's clock (M2 design 3.13).
INSERT INTO users (id, email, password, display_name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

-- name: GetUser :one
SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
FROM users
WHERE id = sqlc.arg(id);

-- name: FindLoginAccount :one
-- What login reads before its transaction; the hash is its snapshot (M2 design 3.5).
SELECT id, password
FROM users
WHERE email = sqlc.arg(email);

-- name: LockUserForCredentials :one
-- The account row lock of M2 design 3.5. FOR NO KEY UPDATE conflicts with itself and with
-- FOR UPDATE, so the credential transactions of one account run one after another; it does not
-- conflict with the FOR KEY SHARE that foreign-key checks take, so inserting rows that reference
-- the account does not wait.
SELECT password, is_active
FROM users
WHERE id = sqlc.arg(id)
FOR NO KEY UPDATE;

-- name: UpdateUser :one
-- PATCH /me: only the fields that are set change (M2 design 3.14); the rest keep what a concurrent write left.
UPDATE users
SET updated_at    = sqlc.arg(now),
    first_name    = CASE WHEN sqlc.arg(set_first_name)::boolean THEN sqlc.arg(first_name)::text ELSE first_name END,
    last_name     = CASE WHEN sqlc.arg(set_last_name)::boolean THEN sqlc.arg(last_name)::text ELSE last_name END,
    display_name  = CASE WHEN sqlc.arg(set_display_name)::boolean THEN sqlc.arg(display_name)::text ELSE display_name END,
    user_timezone = CASE WHEN sqlc.arg(set_user_timezone)::boolean THEN sqlc.arg(user_timezone)::text ELSE user_timezone END
WHERE id = sqlc.arg(id)
RETURNING id, email, first_name, last_name, display_name, user_timezone, created_at;

-- name: UpdatePasswordHash :exec
UPDATE users
SET password = sqlc.arg(password), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
