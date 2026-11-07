-- name: DeleteProjectCycles :many
-- The deletion of a deleted project's planning rows, a step per table (M6 P1 spec 2.4, P2 spec 2.4 and R16), here its
-- cycles: a batch of the project's undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they
-- come and soft-deleted at the moment and by the account given. The caller runs it again from the batch's greatest id
-- until a batch finds none (a row deleted while the batch waits for it is passed over; each batch reads a new
-- snapshot). The primary key's index serves the order and bound.
UPDATE cycles AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM cycles r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectModules :many
-- DeleteProjectCycles' batch, of the project's modules, archived ones too.
UPDATE modules AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM modules r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectModuleMembers :many
-- DeleteProjectCycles' batch, of the project's modules' members.
UPDATE module_members AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM module_members r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectModuleLinks :many
-- DeleteProjectCycles' batch, of the project's modules' links.
UPDATE module_links AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM module_links r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: ProjectDeletionStepRemaining :one
-- Candidate-only read of one step at a budget boundary: no lock, no write.
SELECT CASE sqlc.arg(step)::text
WHEN 'cycles' THEN EXISTS (SELECT 1 FROM cycles r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'modules' THEN EXISTS (SELECT 1 FROM modules r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'module_members' THEN EXISTS (SELECT 1 FROM module_members r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'module_links' THEN EXISTS (SELECT 1 FROM module_links r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
ELSE false END AS remaining;
