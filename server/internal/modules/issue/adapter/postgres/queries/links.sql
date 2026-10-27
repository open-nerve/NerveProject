-- name: GetLink :one
SELECT id, workspace_id, project_id, issue_id, url, title, metadata, created_by_id, created_at, updated_at
FROM issue_links WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: Links :many
SELECT id, workspace_id, project_id, issue_id, url, title, metadata, created_by_id, created_at, updated_at
FROM issue_links WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id DESC;

-- name: LinkURLExists :one
SELECT EXISTS (SELECT 1 FROM issue_links
    WHERE issue_id = sqlc.arg(issue_id) AND url = sqlc.arg(url)
        AND id <> sqlc.arg(exclude_id) AND deleted_at IS NULL);

-- name: InsertLink :exec
INSERT INTO issue_links (id, workspace_id, project_id, issue_id, url, title,
    created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id), sqlc.arg(url), sqlc.narg(title),
    sqlc.narg(by)::uuid, sqlc.narg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: UpdateLink :one
UPDATE issue_links SET url = sqlc.arg(url), title = sqlc.narg(title),
    metadata = CASE WHEN url <> sqlc.arg(url) THEN '{}'::jsonb ELSE metadata END,
    updated_by_id = sqlc.narg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING id, workspace_id, project_id, issue_id, url, title, metadata, created_by_id, created_at, updated_at;

-- name: DeleteLink :exec
UPDATE issue_links SET deleted_at = sqlc.arg(at)::timestamptz,
    updated_by_id = sqlc.narg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
