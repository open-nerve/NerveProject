-- name: CycleTargets :many
-- Of the cycles ids, the project's undeleted ones, archived or not, in id order, each with its name and whether it is
-- archived: the cycle a work item's write names (M6 P3a spec R1), read under the project's lock the write holds, which
-- every write on the project's cycles takes first. Another project's, a deleted one and an id of none are not among
-- them. The primary key serves the ids.
SELECT id, name, (archived_at IS NOT NULL)::boolean AS archived
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id;

-- name: ModuleTargets :many
-- Of the modules ids, the project's undeleted ones, archived or not, in id order, each with its name and whether it is
-- archived: the modules a work item's write names (M6 P3a spec R1), read as CycleTargets reads the cycles.
SELECT id, name, (archived_at IS NOT NULL)::boolean AS archived
FROM modules
WHERE project_id = sqlc.arg(project_id) AND id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id;

-- name: OpenCycleIDs :many
-- The project's undeleted cycles, archived or not, that have not ended on today, the project's day, in id order: those
-- without dates (a cycle has both or neither, cycles_dates_check) or ending on today or later, whose status on today
-- is not completed (domain.StatusOf: completed is after the end day). Their work items are not archived automatically
-- (M6 P3a spec R9, design 7). The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves
-- it.
SELECT id
FROM cycles
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND (end_date IS NULL OR end_date >= sqlc.arg(today)::date)
ORDER BY id;

-- name: OpenModuleIDs :many
-- The project's undeleted modules, archived or not and in any status, whose target day has not passed on today, the
-- project's day, in id order: those without one or with one on today or later. Their work items are not archived
-- automatically (M6 P3a spec R9, design 7, as Plane). The list index (project_id, sort_order, created_at, id) WHERE
-- deleted_at IS NULL serves it.
SELECT id
FROM modules
WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL AND (target_date IS NULL OR target_date >= sqlc.arg(today)::date)
ORDER BY id;
