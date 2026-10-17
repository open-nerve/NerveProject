-- name: LockIssueProject :one
-- ProjectGate.Lock, what project.Provide offers the issue module (M4 design 6.5): the first lock of a write on the
-- work-item side (3.5, decision point 1: B), the project's row FOR NO KEY UPDATE, its workspace's row not locked.
-- After a wait Postgres evaluates deleted_at IS NULL again on the row's newest version, so a project deleted meanwhile
-- reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, default_assignee_id
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;

-- name: NextIssueSequence :one
-- Sequences.Next (M4 design 6.5), under the project's FOR NO KEY UPDATE: the project's next work-item number. The
-- counter never goes back, so a number is never used again (3.7).
UPDATE projects
SET last_issue_sequence = last_issue_sequence + 1
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
RETURNING last_issue_sequence;

-- name: IssueStates :many
-- StateCatalog (M4 design 6.5): the project's undeleted states, its triage state too, in id order. Read under the
-- project's FOR NO KEY UPDATE, which every write of a state holds too (M3 design 3.6), it stays as read (M4 design 3.5).
SELECT id, "group", "default"
FROM states
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY id;

-- name: IssueLabels :many
-- LabelCatalog (M4 design 6.5): those of ids that are the project's undeleted labels, in id order; stable under the
-- project's FOR NO KEY UPDATE, as IssueStates.
SELECT id
FROM labels
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND id = ANY (sqlc.arg(ids)::uuid[])
ORDER BY id;

-- name: ActiveRoles :many
-- ProjectMembers (M4 design 6.5): the active memberships of the project of those of the accounts ids that have one,
-- and their roles; stable under the project's FOR NO KEY UPDATE, as IssueStates.
SELECT member_id, role
FROM project_members
WHERE project_id = sqlc.arg(project_id) AND member_id = ANY (sqlc.arg(ids)::uuid[]) AND is_active AND deleted_at IS NULL
ORDER BY member_id;
