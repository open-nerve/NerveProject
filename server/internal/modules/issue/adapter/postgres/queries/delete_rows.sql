-- name: DeleteProjectIssueRows :many
-- The deletion of a project's work-item side (M4 design 3.6), the work items' table: a batch of the project's
-- undeleted rows after the id `after`, in id order, locked FOR NO KEY UPDATE as they come and soft-deleted at the
-- moment and by the account given (convention 5: two batches over the same rows take them in one order). The caller
-- runs the tables' statements in the global order (3.5) and, for each table, again from the batch's greatest id until
-- a batch finds none (a row deleted while the batch waits for it is passed over and the next one taken; each batch
-- reads a new snapshot). The primary key's index serves the id order and the bound. It returns the ids it deleted.
UPDATE issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectAssigneeRows :many
-- DeleteProjectIssueRows' batch, of the work items' assignees.
UPDATE issue_assignees AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_assignees r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectLabelRows :many
-- DeleteProjectIssueRows' batch, of the work items' labels.
UPDATE issue_labels AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_labels r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectLinkRows :many
-- DeleteProjectIssueRows' batch, of the work items' links.
UPDATE issue_links AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_links r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectSubscriberRows :many
-- DeleteProjectIssueRows' batch, of the work items' subscribers.
UPDATE issue_subscribers AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_subscribers r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectCommentRows :many
-- DeleteProjectIssueRows' batch, of the work items' comments.
UPDATE issue_comments AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_comments r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectReactionRows :many
-- DeleteProjectIssueRows' batch, of the work items' reactions.
UPDATE issue_reactions AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_reactions r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectCommentReactionRows :many
-- DeleteProjectIssueRows' batch, of the work items' reactions.
UPDATE comment_reactions AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM comment_reactions r
    WHERE r.comment_id IN (SELECT c.id FROM issue_comments c WHERE c.project_id = sqlc.arg(project_id)) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectActivityRows :many
-- DeleteProjectIssueRows' batch, of the work items' activity.
UPDATE issue_activities AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_activities r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectMentionRows :many
-- DeleteProjectIssueRows' batch, of the work items' subscribers.
UPDATE issue_mentions AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_mentions r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;


-- name: DeleteProjectDescriptionVersionRows :many
-- DeleteProjectIssueRows' batch, of the work items' subscribers.
UPDATE issue_description_versions AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_description_versions r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteProjectDraftRows :many
-- Drafts precede issues in the shared table order; only live drafts are stamped.
UPDATE draft_issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM draft_issues r
    WHERE r.project_id = sqlc.arg(project_id) AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;

-- name: DeleteWorkspaceDraftRows :many
-- Project drafts are handled by their project's cascade; preserve old deletions.
UPDATE draft_issues AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM draft_issues r
    WHERE r.workspace_id = sqlc.arg(workspace_id) AND r.project_id IS NULL AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE)
RETURNING t.id;
