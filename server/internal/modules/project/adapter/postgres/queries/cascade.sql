-- name: DeleteProjects :exec
-- The first step of deleting projects (M3 design 3.3, 3.6), one statement (convention 5) under the parent's FOR NO KEY
-- UPDATE, which the caller took: the workspace's, when it deletes the workspace, and the project's, when it deletes
-- the project. The workspace's undeleted projects, archived ones too, or only the one project_id names when it is
-- given, at the moment and by the account of the deletion. Projects deleted before keep their moment.
UPDATE projects
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteProjectMembers :exec
-- The memberships of those projects, active and ended ones alike.
UPDATE project_members
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteProjectPreferences :exec
UPDATE project_user_properties
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

-- name: DeleteStates :exec
-- The triage states too.
UPDATE states
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND (sqlc.narg(project_id)::uuid IS NULL OR project_id = sqlc.narg(project_id))
  AND deleted_at IS NULL;

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

-- name: LockActiveMemberProjects :many
-- The first step of ending an account's project memberships, EndMemberships' (M3 design 3.6 convention 6): run when
-- it is called, after the caller ended his membership of the workspaces, it finds their undeleted projects, archived
-- ones too, in which he has an active membership, and locks them FOR NO KEY UPDATE in id order. The lock is taken as
-- the sorted rows come, so the order is the ids'. After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version: a project deleted meanwhile is left out.
SELECT p.id
FROM projects p
WHERE p.workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND p.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM project_members m
              WHERE m.project_id = p.id AND m.member_id = sqlc.arg(member_id) AND m.is_active AND m.deleted_at IS NULL)
ORDER BY p.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
-- The second step, under the projects' locks: whether the account is the only active admin of one of the projects
-- that has another active member, whom ending his membership would leave without an admin (M3 design 3.7 rule 2). A
-- project where he is alone, or that has another active admin, does not count.
SELECT EXISTS (
    SELECT 1 FROM project_members m
    WHERE m.project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND m.member_id = sqlc.arg(member_id) AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM project_members a
                      WHERE a.project_id = m.project_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM project_members o
                  WHERE o.project_id = m.project_id AND o.member_id <> m.member_id AND o.is_active AND o.deleted_at IS NULL));

-- name: EndMemberships :exec
-- The last step, one statement under the projects' locks (convention 5): the account's active memberships of the
-- projects end, at the moment and by the account given; the rows stay, and an ended or deleted one keeps its columns.
UPDATE project_members
SET is_active = false, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(ended_by)::uuid
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active AND deleted_at IS NULL;
