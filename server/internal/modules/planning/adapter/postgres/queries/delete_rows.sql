-- name: DeleteProjectCycles :many
-- The deletion of a deleted project's cycles (M6 P1 spec 2.4): a batch of the project's undeleted cycles after the
-- id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the moment and by the account
-- given. The caller runs it again from the batch's greatest id until a batch finds none (a row deleted while the batch
-- waits for it is passed over; each batch reads a new snapshot). The primary key's index serves the order and bound.
UPDATE cycles AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM cycles r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: ProjectCyclesRemaining :one
-- Candidate-only read at a budget boundary: no lock, no write.
SELECT EXISTS (SELECT 1 FROM cycles r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid) AS remaining;
