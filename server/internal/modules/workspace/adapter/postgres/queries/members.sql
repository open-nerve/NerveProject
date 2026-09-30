-- name: CreateMember :exec
INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: ListMembers :many
-- listWorkspaceMembers: every undeleted membership, active or not, by the time it began (M3 design 3.12).
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY created_at, id;

-- name: MemberByID :one
-- updateWorkspaceMember reads the membership before the workspace's lock, for the workspace, and again under it.
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: MemberOf :one
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE: the user's undeleted membership, active or
-- ended; the partial unique index holds at most one.
SELECT id, workspace_id, member_id, role, is_active, created_at
FROM workspace_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL;

-- name: RestoreMember :exec
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE: an ended membership active again, with the
-- invitation's role (M3 design 3.8).
UPDATE workspace_members
SET is_active = true, role = sqlc.arg(role), updated_by_id = sqlc.arg(restored_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);

-- name: UpdateMemberRole :one
-- updateWorkspaceMember, under the workspace's FOR NO KEY UPDATE.
UPDATE workspace_members
SET role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING id, workspace_id, member_id, role, is_active, created_at;

-- name: DeleteWorkspaceMembers :exec
-- deleteWorkspace's cascade: every undeleted membership of the workspace, active or not, one statement in scan order
-- under the workspace's FOR NO KEY UPDATE (M3 design 3.6 convention 5). A row deleted before keeps its time.
UPDATE workspace_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: ActiveRole :one
-- WorkspaceRoles (M3 design 6.5): the user's role when his membership is active, its row not deleted and
-- the workspace not deleted. The partial unique index holds at most one undeleted row per pair.
SELECT m.role
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id
WHERE m.workspace_id = sqlc.arg(workspace_id) AND m.member_id = sqlc.arg(user_id)
  AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL;
