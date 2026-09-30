-- name: CreateWorkspace :one
-- The audit columns come from the use case's clock (M2 design 3.13); RETURNING gives the values as stored.
INSERT INTO workspaces (id, name, slug, organization_size, timezone, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(name), sqlc.arg(slug), sqlc.narg(organization_size), sqlc.arg(timezone),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, name, slug, organization_size, timezone, created_at, updated_at;

-- name: ListWorkspaces :many
-- listWorkspaces (M3 design 3.12): the undeleted workspaces of which the user is an active member, by name,
-- then id, with his role and the number of active members.
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at, m.role,
       (SELECT count(*) FROM workspace_members c
        WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w
JOIN workspace_members m ON m.workspace_id = w.id
WHERE m.member_id = sqlc.arg(user_id) AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL
ORDER BY w.name, w.id;

-- name: WorkspaceBySlug :one
SELECT w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
       (SELECT count(*) FROM workspace_members c
        WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members
FROM workspaces w
WHERE w.slug = sqlc.arg(slug) AND w.deleted_at IS NULL;

-- name: UpdateWorkspace :one
-- updateWorkspace, under the workspace's FOR NO KEY UPDATE: only the fields that are set change (M2 design 3.14).
-- RETURNING gives the values as stored and the number of active members.
UPDATE workspaces w
SET name              = CASE WHEN sqlc.arg(set_name)::boolean THEN sqlc.arg(name)::text ELSE w.name END,
    organization_size = CASE WHEN sqlc.arg(set_organization_size)::boolean THEN sqlc.arg(organization_size)::text
                        ELSE w.organization_size END,
    timezone          = CASE WHEN sqlc.arg(set_timezone)::boolean THEN sqlc.arg(timezone)::text ELSE w.timezone END,
    updated_by_id     = sqlc.arg(updated_by),
    updated_at        = sqlc.arg(now)
WHERE w.id = sqlc.arg(id)
RETURNING w.id, w.name, w.slug, w.organization_size, w.timezone, w.created_at, w.updated_at,
          (SELECT count(*) FROM workspace_members c
           WHERE c.workspace_id = w.id AND c.is_active AND c.deleted_at IS NULL) AS total_members;

-- name: DeleteWorkspace :exec
-- deleteWorkspace's first step, under the workspace's FOR NO KEY UPDATE: the slug is free again at once (the partial
-- unique index). The rows under it are soft-deleted at the same moment by the steps that follow (M3 design 3.6).
UPDATE workspaces
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: SlugTaken :one
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL);

-- name: LockWorkspaceBySlug :one
-- The parent lock of a write that changes the workspace row itself or a membership (M3 design 3.6 convention 2):
-- FOR NO KEY UPDATE waits for another FOR NO KEY UPDATE and for FOR SHARE. After a wait, Postgres evaluates
-- deleted_at IS NULL again on the row's newest version, so a workspace deleted meanwhile reads no row.
SELECT id
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: ShareWorkspaceBySlug :one
-- The parent lock of a write that adds or changes a row under the workspace (M3 design 3.6 convention 2): FOR SHARE
-- does not wait for another FOR SHARE, and it holds off the workspace's deletion, which the FOR KEY SHARE of a
-- foreign key check does not.
SELECT id
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR SHARE;
