-- name: CycleNames :many
-- Of the projects, every undeleted cycle, archived or not, with its name, in id order: the names a list of work items
-- ordered by the cycles' names keys them by (M6 P3b spec R2), read in the list's read-only snapshot. Another project's
-- and a deleted one are not among them, nor is anything when no project is asked about. The projects' toggles do not
-- matter (M6 design 3.8). The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT id, name
FROM cycles
WHERE project_id = ANY(sqlc.arg(project_ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id;

-- name: ModuleNames :many
-- Of the projects, every undeleted module, archived or not and in any status, with its name, in id order: the names a
-- list of work items ordered by the modules' names keys them by (M6 P3b spec R2), read as CycleNames reads the cycles'.
-- The list index (project_id, sort_order, created_at, id) WHERE deleted_at IS NULL serves it.
SELECT id, name
FROM modules
WHERE project_id = ANY(sqlc.arg(project_ids)::uuid[]) AND deleted_at IS NULL
ORDER BY id;
