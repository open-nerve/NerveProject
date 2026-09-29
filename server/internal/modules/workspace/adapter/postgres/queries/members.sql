-- name: CreateMember :exec
INSERT INTO workspace_members (id, workspace_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(member_id), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: ActiveRole :one
-- WorkspaceRoles (M3 design 6.5): the user's role when his membership is active, its row not deleted and
-- the workspace not deleted. The partial unique index holds at most one undeleted row per pair.
SELECT m.role
FROM workspace_members m
JOIN workspaces w ON w.id = m.workspace_id
WHERE m.workspace_id = sqlc.arg(workspace_id) AND m.member_id = sqlc.arg(user_id)
  AND m.is_active AND m.deleted_at IS NULL AND w.deleted_at IS NULL;
