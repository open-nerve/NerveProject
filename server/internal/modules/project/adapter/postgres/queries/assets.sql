-- name: AssetProjectArchived :one
SELECT (archived_at IS NOT NULL)::boolean AS archived FROM projects WHERE id=sqlc.arg(id) AND deleted_at IS NULL;

-- name: AssetProjectFacts :many
SELECT id,deleted_at,updated_by_id AS deleted_by_id FROM projects WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id;
