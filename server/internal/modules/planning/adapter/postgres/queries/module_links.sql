-- name: GetModuleLink :one
-- The undeleted module link id (M6 P2 spec 2.3), read in the caller's transaction: under the project's lock for a
-- write, which every write on the project's modules and their links takes first, so it stays as read until the write
-- commits. Its module is read apart (GetModule).
SELECT id, workspace_id, project_id, module_id, url, title, metadata, created_by_id, created_at, updated_at
FROM module_links
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ListModuleLinks :many
-- The module's undeleted links, the latest created first, then by id descending (M6 P2 spec 2.3, as a work item's).
-- module_links_module_id_idx (module_id) serves it.
SELECT id, workspace_id, project_id, module_id, url, title, metadata, created_by_id, created_at, updated_at
FROM module_links
WHERE module_id = sqlc.arg(module_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC;

-- name: InsertModuleLink :exec
-- A new link of a module, its metadata empty, written at one moment by one account. A URL another undeleted link of the
-- module has, as it is spelt, breaks module_links_module_id_url_key.
INSERT INTO module_links (id, workspace_id, project_id, module_id, url, title, created_by_id, updated_by_id, created_at,
                          updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(module_id), sqlc.arg(url), sqlc.narg(title),
        sqlc.narg(by)::uuid, sqlc.narg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: UpdateModuleLink :one
-- A change of a link: its URL and title, last changed by one account at one moment. A URL it did not have empties its
-- metadata; one it had keeps the metadata as it is now, which a crawl may have written since the link was read (M6 P2
-- spec R11). The link as written.
UPDATE module_links
SET url           = sqlc.arg(url),
    title         = sqlc.narg(title),
    metadata      = CASE WHEN url <> sqlc.arg(url) THEN '{}'::jsonb ELSE metadata END,
    updated_by_id = sqlc.narg(by)::uuid,
    updated_at    = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, workspace_id, project_id, module_id, url, title, metadata, created_by_id, created_at, updated_at;

-- name: DeleteModuleLink :exec
-- A link soft-deleted by one account at one moment, its last change the same; one deleted already keeps its deletion.
UPDATE module_links
SET deleted_at    = sqlc.arg(at)::timestamptz,
    updated_at    = sqlc.arg(at),
    updated_by_id = sqlc.arg(by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
