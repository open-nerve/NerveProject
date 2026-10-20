-- name: DeleteIssueRow :exec
-- deleteIssue's work item (M4 design 3.6), under its project's FOR NO KEY UPDATE, read again under it, archived or not:
-- soft-deleted at the moment and by the account given. Every write of a work item's rows holds its project's lock first
-- (3.5), so the row is as the use case read it; it is not locked by id first (M3 design 3.6, convention 5).
UPDATE issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueAssigneeRows :exec
-- The deleted work item's assignees (M4 design 3.6): its undeleted rows, soft-deleted at the moment and by the account
-- given, in one statement, which locks them in its scan's order. Every write of these rows holds the project's FOR NO
-- KEY UPDATE first (3.5): no other write reaches them meanwhile, so they are not locked by id first (M3 design 3.6,
-- convention 5). A row deleted before keeps its deletion. The index on issue_id serves it.
UPDATE issue_assignees
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueLabelRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's labels.
UPDATE issue_labels
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueSubscriberRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's subscribers.
UPDATE issue_subscribers
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;

-- name: DeleteIssueActivityRows :exec
-- DeleteIssueAssigneeRows' statement, of the deleted work item's activity (3.6: it goes with its work item); the index
-- on (issue_id, created_at, id) serves it.
UPDATE issue_activities
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;
