-- name: GetCycle :one
-- The undeleted cycle id, archived or not (M6 P1 spec 2.3), read in the caller's transaction: under the project's lock
-- for a write, which every write on the project's cycles takes first, so it stays as read until the write commits.
SELECT id, workspace_id, project_id, name, description, start_date, end_date, owned_by_id, sort_order, archived_at,
       created_at, updated_at, created_by_id, updated_by_id
FROM cycles
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: ProjectCycleDates :many
-- The dates of the project's undeleted cycles, archived ones too: the overlap check reads them under the project's lock
-- (M6 design 3.1). The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT id, start_date, end_date
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL;

-- name: LowestCycleSortOrder :one
-- The lowest sort order of the project's undeleted cycles, archived ones too, no row when it has none: a new cycle goes
-- before it (M6 P1 spec 2.2). The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT sort_order
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL
ORDER BY sort_order
LIMIT 1;

-- name: CreateCycle :exec
-- A new cycle, written at one moment by one account.
INSERT INTO cycles (id, workspace_id, project_id, name, description, start_date, end_date, owned_by_id, sort_order,
                    created_at, updated_at, created_by_id, updated_by_id)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(name), sqlc.arg(description),
        sqlc.narg(start_date), sqlc.narg(end_date), sqlc.narg(owned_by_id), sqlc.arg(sort_order),
        sqlc.arg(created_at), sqlc.arg(updated_at), sqlc.narg(created_by_id), sqlc.narg(updated_by_id));

-- name: ListCycles :many
-- The project's undeleted, unarchived cycles in their default order (M6 P1 spec 2.3): by sort order, then creation,
-- then id. The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT id, workspace_id, project_id, name, description, start_date, end_date, owned_by_id, sort_order, archived_at,
       created_at, updated_at, created_by_id, updated_by_id
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND archived_at IS NULL
ORDER BY sort_order, created_at, id;

-- name: ListArchivedCycles :many
-- The project's undeleted, archived cycles alone, the latest archived first, then by id (M6 P1 spec 2.3, R9).
SELECT id, workspace_id, project_id, name, description, start_date, end_date, owned_by_id, sort_order, archived_at,
       created_at, updated_at, created_by_id, updated_by_id
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND archived_at IS NOT NULL
ORDER BY archived_at DESC, id;
