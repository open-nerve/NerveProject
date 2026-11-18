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
-- not locked by id first (M3 design 3.6, convention 5). A subscription's FOR SHARE of one of them (ShareIssue, M7 P1a
-- spec 2.4), which holds no project lock, makes the statement wait until the subscription ends. A row deleted before
-- keeps its deletion.
UPDATE issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueIntakeRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' intake rows (M7 P2a spec 2.10): a work item's intake row
-- goes with it, at its moment, whatever its status. Another row that names it as its duplicate stays as it is and reads
-- as no target (spec 2.8). Every write of an intake row holds the project's FOR NO KEY UPDATE first (spec 2.4, 2.5).
-- The index on issue_id serves it.
UPDATE intake_issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

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

-- name: DeleteIssueCycleRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work items' cycle memberships (M6 P3a spec 2.5): they go with
-- their work item, at its moment, and no membership activity is written. The index on issue_id serves it.
UPDATE cycle_issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueModuleRows :exec
-- DeleteIssueCycleRows' statement, of the deleted work items' module memberships.
UPDATE module_issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND deleted_at IS NULL;

-- name: DeleteIssueLinkRows :exec
-- Lock the subtree's live links in id order before stamping them; earlier deletions keep their audit.
UPDATE issue_links AS t
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE t.id IN (
    SELECT r.id FROM issue_links r
    WHERE r.issue_id = ANY (sqlc.arg(issue_ids)::uuid[]) AND r.deleted_at IS NULL
    ORDER BY r.id FOR NO KEY UPDATE);

-- name: DeleteIssueSubscriberRows :exec
-- The deleted work items' subscribers: their undeleted rows, soft-deleted at the moment and by the account given, in one
-- statement, which locks them in its scan's order. Not every write of these rows holds the project's FOR NO KEY UPDATE:
-- subscribeIssue and unsubscribeIssue hold the work item's row FOR SHARE instead (ShareIssue, M7 P1a spec 2.4). The
-- deletion holds both locks: the project's, and each deleted work item's row since DeleteIssueRows, which a FOR SHARE
-- waits for. A subscription that held the row first has committed before DeleteIssueRows went on, and this later
-- statement reads its row; one that waited finds the work item deleted and writes nothing. So no other write reaches
-- these rows meanwhile, and they are not locked by id first (M3 design 3.6, convention 5); InsertSubscribers gives the
-- rows' lock order. A row deleted before keeps its deletion. The index on issue_id serves it.
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
