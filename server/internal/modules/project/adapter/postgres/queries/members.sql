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

-- name: ListMembers :many
-- listProjectMembers (M3 design 3.12, 5.2): the project's active undeleted memberships, in the order they were made,
-- then by id. It reads project_members alone: an active member of a project stays an active member of its workspace,
-- for every growth locks his workspace membership and every shrinking ends his project memberships (3.6 conventions 3
-- and 6).
SELECT id, project_id, member_id, role, created_at
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND is_active AND deleted_at IS NULL
ORDER BY created_at, id;

-- name: RestoreMember :exec
-- addProjectMembers and joinProject, under the project's FOR NO KEY UPDATE (M3 design 3.6 convention 6): an ended
-- membership active again, with the role the use case gives, at the moment and by the account given; it keeps its id
-- and its created_at.
UPDATE project_members
SET is_active = true, role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by)::uuid, updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id);
