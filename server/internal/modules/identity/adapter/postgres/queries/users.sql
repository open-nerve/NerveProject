-- name: CreateUser :exec
-- The other columns take their defaults; the audit columns come from the use case's clock (M2 design 3.13).
INSERT INTO users (id, email, password, display_name, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(email), sqlc.arg(password), sqlc.arg(display_name), sqlc.arg(now), sqlc.arg(now));

-- name: GetUser :one
SELECT id, email, first_name, last_name, display_name, user_timezone, created_at
FROM users
WHERE id = sqlc.arg(id);
