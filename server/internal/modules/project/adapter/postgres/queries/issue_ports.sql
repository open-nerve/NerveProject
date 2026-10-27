-- name: LockIssueProject :one
-- ProjectGate.Lock, what project.Provide offers the issue module (M4 design 6.5): the first lock of a write on the
-- work-item side (3.5, decision point 1: B), the project's row FOR NO KEY UPDATE, its workspace's row not locked.
-- After a wait Postgres evaluates deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile
-- reads no row. The identifier names the project's work items in their activity (3.13), as it is under the lock.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id, identifier, archive_in
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: LockIssueProjects :many
-- ProjectGate.LockMany (M4 design 6.5): the first locks of a write across projects (convention 7, 3.5), the undeleted
-- projects among ids FOR NO KEY UPDATE, as LockIssueProject, in one statement, in id order. The lock is taken as the
-- sorted rows come, so the order is the ids' (S5 E4). After a wait, Postgres evaluates deleted_at IS NULL again on the
-- row's newest version: a project deleted meanwhile is left out.
SELECT id, workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id, identifier, archive_in
FROM projects
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id
FOR NO KEY UPDATE;

-- name: NextIssueSequence :one
-- Sequences.Next (M4 design 6.5), under the project's FOR NO KEY UPDATE: the project's next work-item number. The
-- counter never goes back, so a number is never used again (3.7).
UPDATE projects
SET last_issue_sequence = last_issue_sequence + 1
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING last_issue_sequence;

-- name: IssueStates :many
-- StateCatalog (M4 design 6.5): the project's undeleted states, its triage state too, in id order, with their names,
-- which an update's activity names a state by (3.13). Read under the project's FOR NO KEY UPDATE, which every write of
-- a state holds too (M3 design 3.6), it stays as read (M4 design 3.5).
SELECT id, name, "group", "default"
FROM states
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY id;

-- name: IssueLabels :many
-- LabelCatalog (M4 design 6.5): those of ids that are the project's undeleted labels, in id order, with their names,
-- which an update's activity names a label by (3.13); stable under the project's FOR NO KEY UPDATE, as IssueStates.
SELECT id, name
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND id = ANY (sqlc.arg(ids)::uuid[])
ORDER BY id;

-- name: ProjectIdentifiers :many
-- ProjectIdentifiers (M4 design 3.7): the workspace's undeleted projects, archived or not, with their identifiers, in id
-- order, read without a lock: the projects a work item's identifier may name. A deleted project's identifier may be
-- another's since (M3 design 3.19), so it names none.
SELECT id, identifier
FROM projects
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY id;

-- name: ActiveRoles :many
-- ProjectMembers (M4 design 6.5): the active memberships of the project of those of the accounts ids that have one,
-- and their roles; stable under the project's FOR NO KEY UPDATE, as IssueStates.
SELECT member_id, role
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(ids)::uuid[]) AND is_active AND deleted_at IS NULL
ORDER BY member_id;

-- name: IssueMentionMembers :many
-- Read active mention candidates and the guest flag in the same statement.
SELECT m.member_id, m.role, (m.role = 5 AND NOT p.guest_view_all_features)::boolean AS own_only
FROM project_members m JOIN projects p ON p.id = m.project_id
WHERE m.project_id = sqlc.arg(project_id) AND m.member_id = ANY (sqlc.arg(ids)::uuid[])
  AND m.is_active AND m.deleted_at IS NULL AND p.deleted_at IS NULL
ORDER BY m.member_id;

-- name: UndeletedIssueProjectIDs :many
-- Personal drafts remain reachable after project exit and archival.
SELECT id FROM projects
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY id;

-- name: ShareIssueProjects :many
-- Draft saves use SHARE, compatible with other saves but excluding project
-- lifecycle and membership writes. ANY deduplicates before ascending row locks.
SELECT id, workspace_id FROM projects
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id
FOR SHARE;

-- name: AllUndeletedIssueProjectIDs :many
-- Retention includes archived projects, without policy or membership filters.
SELECT id FROM projects
WHERE deleted_at IS NULL
ORDER BY id;
