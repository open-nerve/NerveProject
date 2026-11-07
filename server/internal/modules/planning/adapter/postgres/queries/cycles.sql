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

-- name: UpdateCycle :exec
-- A change of a cycle: its name, description, dates and sort order, last changed by one account at one moment. Its
-- owner, archive and creation are not a change's (M6 P1 spec 2.3).
UPDATE cycles
SET name          = sqlc.arg(name),
    description   = sqlc.arg(description),
    start_date    = sqlc.narg(start_date),
    end_date      = sqlc.narg(end_date),
    sort_order    = sqlc.arg(sort_order),
    updated_at    = sqlc.arg(updated_at),
    updated_by_id = sqlc.narg(updated_by_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: SetCycleArchived :exec
-- A cycle archived at one moment, or no longer, last changed by one account at that moment (M6 P1 spec R3).
UPDATE cycles
SET archived_at   = sqlc.narg(archived_at),
    updated_at    = sqlc.arg(updated_at),
    updated_by_id = sqlc.narg(updated_by_id)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteCycle :exec
-- A cycle soft-deleted by one account at one moment, its last change the same (M6 P1 spec 2.4); one deleted already
-- keeps its deletion.
UPDATE cycles
SET deleted_at    = sqlc.arg(at),
    updated_at    = sqlc.arg(at),
    updated_by_id = sqlc.arg(by)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
