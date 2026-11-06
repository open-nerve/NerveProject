-- name: PlanningProjectFacts :one
-- What project.Provide offers the planning module (M6 P1 spec 2.1), read without a lock: the undeleted project's
-- workspace, whether it is archived, its time zone and its cycle and module toggles. A deleted project reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, timezone, cycle_view, module_view
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: LockPlanningProject :one
-- PlanningProjectFacts under the first lock of a planning write (M6 design 6): the project's row FOR NO KEY UPDATE, as
-- LockIssueProject takes it, its workspace's row not locked. After a wait Postgres evaluates deleted_at IS NULL again on
-- the row's newest version, so a project deleted meanwhile reads no row.
SELECT workspace_id, (archived_at IS NOT NULL)::boolean AS archived, timezone, cycle_view, module_view
FROM projects
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR NO KEY UPDATE;
