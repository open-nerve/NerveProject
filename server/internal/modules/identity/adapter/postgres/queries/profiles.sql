-- name: CreateProfile :exec
-- Every preference takes its default (Plane's model defaults, M2 design 4.3).
INSERT INTO profiles (id, user_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(user_id), sqlc.arg(now), sqlc.arg(now));
