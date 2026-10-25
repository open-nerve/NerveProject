-- name: IssueSubtree :many
-- deleteIssue's work items (M4 design 3.6, 3.9): the undeleted work item and each undeleted work item under it, however
-- deep, archived ones too, in id order. Read under the project's FOR NO KEY UPDATE, which every write of a parent holds
-- (3.5), so it stays as read until the deletion commits; a work item's parent is of its project, so its subtree is too.
-- UNION reads each work item once. The index on parent_id serves it.
WITH RECURSIVE subtree (id) AS (
    SELECT i.id FROM issues i WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL
    UNION
    SELECT c.id FROM issues c JOIN subtree s ON c.parent_id = s.id WHERE c.deleted_at IS NULL
)
SELECT id FROM subtree ORDER BY id;

-- name: DeleteIssueRows :exec
-- deleteIssue's work items, its subtree (M4 design 3.6), under their project's FOR NO KEY UPDATE, archived or not:
-- soft-deleted at the moment and by the account given, in one statement, which locks them in its scan's order. Every
-- write of a work item's rows holds its project's lock first (3.5), so the rows are as the use case read them; they are
-- not locked by id first (M3 design 3.6, convention 5). A row deleted before keeps its deletion.
UPDATE issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueAssigneeRows :exec
-- The deleted work items' assignees (M4 design 3.6): their undeleted rows, soft-deleted at the moment and by the account
-- given, in one statement, which locks them in its scan's order. Every write of these rows holds the project's FOR NO
-- KEY UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). A row deleted before keeps its deletion. The index on issue_id serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueLabelRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' labels.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueRelationRows :exec
-- The deleted work items' relations (M4 design 3.5, 3.6), both ways: the undeleted rows that name one of them at either
-- end, the other end in this project or in another, soft-deleted at the moment and by the account given, in one
-- statement. A relation with a work item of another project is reached by that project's writes without this
-- project's lock: its deletion, and deleting the work item at that end. Two writers that each took such rows in their
-- scans' order could each hold one the other waits for, so the statement locks its rows FOR NO KEY UPDATE in id order
-- before it writes them, as each of the others does (convention 5's exception, 3.5). A row another deleted while the
-- statement waited for it is evaluated again on its newest version and passed over, and keeps that deletion.
UPDATE issue_relations AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_relations r
    WHERE (r.issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) OR r.related_issue_id = ANY (sqlc.arg(issue_ids)::uuid[]))
        AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteIssueSubscriberRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' subscribers.
UPDATE issue_subscribers
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueCommentRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' comments.
UPDATE issue_comments
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueReactionRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' reactions.
UPDATE issue_reactions
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueCommentReactionRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' reactions.
UPDATE comment_reactions
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE comment_id IN (SELECT id FROM issue_comments WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[])) AND deleted_at IS NULL;

-- name: DeleteIssueActivityRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' activity (3.6: it goes with its work item); the index
-- on (issue_id, created_at, id) serves it.
UPDATE issue_activities
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueMentionRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' subscribers.
UPDATE issue_mentions
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;


-- name: DeleteIssueDescriptionVersionRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' subscribers.
UPDATE issue_description_versions
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;
