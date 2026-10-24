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

-- name: DeleteProjectRelationRows :many
-- DeleteProjectIssueRows' batch, of the relations of the project's work items, both ways (M4 design 3.5, 3.6): the
-- project's rows (project_id is the issue_id end's) and another project's rows whose related_issue_id is a work item
-- of this project, deleted or not: the work items' batches have deleted them before this one. Another project's
-- writes reach such a row without this project's lock (its deletion, deleting its end's work item); every writer of
-- these rows takes them in id order (convention 5's exception, 3.5), as the batch's lock does. The indexes of the two
-- ends serve it, an arm of the union each: the project's rows by project_id, those toward its work items by issues'
-- project_id, then related_issue_id; the primary key's index keeps the undeleted ones after the bound, and a sort
-- puts them in id order for the lock. Each batch so reads all the project's relations both ways and its work items,
-- not every project's relations (a custom plan reads the whole table only where the planner estimates that cheaper).
-- Until P5 sets plan_cache_mode = force_custom_plan (M4 design 3.16), a generic plan may instead merge against the
-- primary key from the bound, reading every project's rows after it (P3 spec, section 7).
UPDATE issue_relations AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_relations r
    WHERE r.id IN (SELECT o.id FROM issue_relations o WHERE o.project_id = sqlc.arg(project_id)
                   UNION ALL
                   SELECT x.id FROM issue_relations x JOIN issues i ON i.id = x.related_issue_id
                   WHERE i.project_id = sqlc.arg(project_id))
        AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid
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
