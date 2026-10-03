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

-- name: CountInactiveMemberships :one
-- ProjectMembershipCounts, for reactivate-member's report (M3 design 3.11, 6.5): the account's ended undeleted
-- memberships of the workspace's projects, read without a lock. A deleted project's memberships are deleted with it.
SELECT count(*)
FROM project_members
WHERE workspace_id = sqlc.arg(workspace_id) AND member_id = sqlc.arg(member_id) AND NOT is_active AND deleted_at IS NULL;

-- name: MemberByID :one
-- A write on a project membership named by its id (M3 design 3.6 convention 2): the undeleted membership, active or
-- ended, read first without a lock for its project and the project's workspace, then again under their locks.
SELECT id, workspace_id, project_id, member_id, role, is_active
FROM project_members
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateMemberRole :one
-- updateProjectMember, under the project's FOR NO KEY UPDATE (M3 design 3.5, 3.6): the active membership's new role,
-- at the moment and by the account given. An ended or deleted one is not written.
UPDATE project_members
SET role = sqlc.arg(role), updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(updated_by)::uuid
WHERE id = sqlc.arg(id) AND is_active AND deleted_at IS NULL
RETURNING id, project_id, member_id, role, created_at;

-- name: EndMember :execrows
-- removeProjectMember and leaveProject, under the project's FOR NO KEY UPDATE (M3 design 3.5, 3.7): the account's
-- active membership of the project ends, at the moment and by the account given; the row stays, its role too.
UPDATE project_members
SET is_active = false, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(ended_by)::uuid
WHERE project_id = sqlc.arg(project_id) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;

-- name: HasOtherAdmin :one
-- leaveProject, under the project's FOR NO KEY UPDATE (M3 design 3.7 rule 1): whether the project has an active admin
-- other than the account.
SELECT EXISTS (SELECT 1 FROM project_members
               WHERE project_id = sqlc.arg(project_id) AND member_id <> sqlc.arg(member_id) AND role = 20 AND is_active
                 AND deleted_at IS NULL);
