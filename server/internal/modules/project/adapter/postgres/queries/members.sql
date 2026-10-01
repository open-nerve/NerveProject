-- name: CreateMember :exec
INSERT INTO project_members (id, workspace_id, project_id, member_id, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(member_id), sqlc.arg(role), sqlc.arg(created_by),
        sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now));

-- name: Memberships :many
-- The accounts' undeleted memberships of the project, active and ended (M3 design 3.5, 3.19): under the project's
-- FOR NO KEY UPDATE, which every change of its memberships takes, they stay as read until the transaction ends.
SELECT id, member_id, role, is_active
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(member_ids)::uuid[]) AND deleted_at IS NULL;
