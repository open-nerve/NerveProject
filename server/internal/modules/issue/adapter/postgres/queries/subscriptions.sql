-- name: ShareIssue :one
-- The undeleted work item, archived or not, locked FOR SHARE in the caller's transaction (M7 P1a spec 2.4): the lock of
-- subscribeIssue and unsubscribeIssue, which take no project lock. Its own row alone, joined to no other table, so that
-- the row is the one lock it takes. FOR SHARE waits for, and is waited for by, every UPDATE of the row (each takes FOR NO
-- KEY UPDATE): the writes that mark the work item changed or write it, and its deletion, DeleteIssueRows and
-- DeleteProjectIssueRows'; not another FOR SHARE, nor a foreign key's FOR KEY SHARE. A work item deleted while it waits
-- is read again in its new version, which deleted_at IS NULL no longer matches: no row.
SELECT id, workspace_id, project_id, created_by_id
FROM issues
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR SHARE;

-- name: RemoveSubscriber :exec
-- unsubscribeIssue's row (M7 P1a spec 2.4): the work item's one undeleted row of the account, soft-deleted at the moment
-- and by the account given, under the work item's FOR SHARE (ShareIssue); none for none. A row deleted before keeps its
-- deletion. The partial unique key on (issue_id, subscriber_id) serves it.
UPDATE issue_subscribers
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND subscriber_id = sqlc.arg(subscriber_id) AND deleted_at IS NULL;

-- name: IsSubscribed :one
-- Whether the account has an undeleted row of the work item's subscribers: the caller's is_subscribed (M7 P1a spec R5),
-- one point read, which the partial unique key on (issue_id, subscriber_id) serves.
SELECT EXISTS (
    SELECT 1 FROM issue_subscribers
    WHERE issue_id = sqlc.arg(issue_id) AND subscriber_id = sqlc.arg(subscriber_id) AND deleted_at IS NULL
) AS subscribed;
