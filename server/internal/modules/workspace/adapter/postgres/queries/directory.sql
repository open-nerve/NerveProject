-- name: DirectoryWorkspace :one
-- WorkspaceDirectory, which workspace.Provide offers the project module (M3 design 6.5): the undeleted workspace with
-- the slug, read without a lock.
SELECT id, timezone
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL;

-- name: ShareDirectoryWorkspace :one
-- WorkspaceDirectory's lock: the parent lock of a write that adds a project under the workspace (M3 design 3.6
-- convention 2), as ShareWorkspaceBySlug takes it. After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version, so a workspace deleted meanwhile reads no row.
SELECT id, timezone
FROM workspaces
WHERE slug = sqlc.arg(slug) AND deleted_at IS NULL
FOR SHARE;

-- name: ShareDirectoryWorkspaceByID :one
-- WorkspaceDirectory's lock by id: the first lock of every write on a project of the workspace (M3 design 3.6
-- convention 2), as ShareWorkspaceByID takes it. FOR SHARE waits for the workspace's FOR NO KEY UPDATE, under which
-- every cascade over its projects runs, and makes it wait; it does not wait for another write on a project. After a
-- wait, Postgres evaluates deleted_at IS NULL again on the row's newest version, so a workspace deleted meanwhile
-- reads no row.
SELECT id, timezone
FROM workspaces
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

-- name: ShareMembers :many
-- WorkspaceMembers (M3 design 3.6 convention 3): the users' undeleted memberships of the workspace, active or not,
-- locked FOR SHARE in id order. The lock is taken as the sorted rows come, so the order is the ids'.
SELECT member_id, role, is_active
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = ANY (sqlc.arg(user_ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id
FOR SHARE;
