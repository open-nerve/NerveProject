-- name: DeleteProjectViews :many
-- The deletion of a deleted project's views (P3b spec 2.7): a batch of its undeleted views after the id `after`, in id
-- order, locked FOR NO KEY UPDATE as they come and soft-deleted at the moment and by the account of the project's
-- deletion. The caller runs it again from the batch's greatest id until a batch finds none (a view deleted while the
-- batch waits for it is passed over; each batch reads a new snapshot).
UPDATE issue_views AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_views r
    WHERE r.project_id = sqlc.arg(project_id)::uuid AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteWorkspaceViews :many
-- DeleteProjectViews' batch, of a deleted workspace's own views, those without a project: its projects' views are
-- their projects' deletions'.
UPDATE issue_views AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_views r
    WHERE r.workspace_id = sqlc.arg(workspace_id) AND r.project_id IS NULL AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
