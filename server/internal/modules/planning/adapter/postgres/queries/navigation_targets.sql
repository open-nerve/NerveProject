-- name: NavigationCycles :many
-- The cycles favorites name (M7 P3c spec 2.4, 2.5), read by the ids asked, without a lock: each undeleted one's
-- workspace, project, name and whether it is archived, another workspace's too. A deleted cycle and an id of none read
-- no row; the projects' toggles do not matter (spec 2.5). The primary key serves it.
SELECT id, workspace_id, project_id, name, (archived_at IS NOT NULL)::boolean AS archived
FROM cycles
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;

-- name: NavigationModules :many
-- The modules favorites name, read as NavigationCycles reads the cycles, in any status.
SELECT id, workspace_id, project_id, name, (archived_at IS NOT NULL)::boolean AS archived
FROM modules
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;
