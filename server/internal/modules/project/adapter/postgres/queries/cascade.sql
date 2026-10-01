-- ProjectCascade's statements (M3 design 3.3, 3.6), each under the workspace's FOR NO KEY UPDATE, which the caller
-- took. The steps of deleting a workspace's projects are one statement each (convention 5): the rows of the workspace
-- not deleted before, at the moment and by the account of the workspace's deletion. Rows deleted before keep their
-- moment.

-- name: DeleteWorkspaceProjects :exec
UPDATE projects
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectMembers :exec
-- Active memberships and ended ones alike.
UPDATE project_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceProjectPreferences :exec
UPDATE project_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: DeleteWorkspaceStates :exec
-- The triage states too.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;

-- name: LockMemberProjects :many
-- The first step of making an account a guest in the workspace's projects, DemoteToGuest's: the workspace's undeleted
-- projects, archived ones too, in which he has an undeleted membership, active or not, FOR NO KEY UPDATE in id order.
-- The lock is taken as the sorted rows come, so the order is the ids'. After a wait, Postgres evaluates deleted_at IS
-- NULL again on the row's newest version: a project deleted meanwhile is left out.
SELECT p.id
FROM projects p
WHERE p.workspace_id = sqlc.arg(workspace_id) AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = sqlc.arg(member_id) AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;

-- name: DemoteMemberships :exec
-- The second step, one statement under the projects' locks (convention 5): the account's undeleted memberships of the
-- projects, active or not, a guest's now, at the moment and by the account given; a guest's keeps its audit columns.
UPDATE project_members
SET role = 5, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(updated_by)::uuid
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND deleted_at IS NULL
  AND role <> 5;
