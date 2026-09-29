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

-- name: SlugTaken :one
SELECT EXISTS (SELECT 1 FROM workspaces WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL);
