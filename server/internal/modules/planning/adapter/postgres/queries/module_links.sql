-- name: GetModuleLink :one
-- The undeleted module link id (M6 P2 spec 2.3), read in the caller's transaction: under the project's lock for a
-- write, which every write on the project's modules and their links takes first, so it stays as read until the write
-- commits. Its module is read apart (GetModule).
SELECT id, workspace_id, project_id, module_id, url, title, metadata, created_by_id, created_at, updated_at
FROM module_links
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
