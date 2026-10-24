-- name: LatestDescriptionVersion :one
SELECT * FROM issue_description_versions
WHERE issue_id = $1 AND deleted_at IS NULL
ORDER BY last_saved_at DESC, id DESC LIMIT 1;

-- name: InsertDescriptionVersion :exec
INSERT INTO issue_description_versions (id,workspace_id,project_id,issue_id,description_html,description_stripped,last_saved_at,owned_by_id,created_by_id,updated_by_id,created_at,updated_at)
VALUES (sqlc.arg(id),sqlc.arg(workspace_id),sqlc.arg(project_id),sqlc.arg(issue_id),sqlc.arg(html),sqlc.arg(text)::text,sqlc.arg(last_saved_at),sqlc.arg(owned_by_id),sqlc.arg(created_by_id)::uuid,sqlc.arg(updated_by_id)::uuid,sqlc.arg(created_at),sqlc.arg(updated_at));

-- name: UpdateDescriptionVersion :exec
UPDATE issue_description_versions SET description_html=sqlc.arg(html),description_stripped=sqlc.arg(text)::text,
 last_saved_at=sqlc.arg(last_saved_at),updated_at=sqlc.arg(updated_at),updated_by_id=sqlc.arg(updated_by_id)::uuid
WHERE id=sqlc.arg(id) AND deleted_at IS NULL;
