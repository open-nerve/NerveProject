-- name: ProjectIssuesRemain :one
-- Whether the project has an undeleted work item, archived or not (M7 P1b spec 2.6, 2.8, R3): the deletion of a deleted
-- project's notifications waits until issue.delete_rows has soft-deleted every one of them, so no fan-out, which holds
-- its work item's row FOR SHARE and reads it undeleted, can write a notification of the project after it. A plain read
-- of what is committed: no lock. The partial index on (project_id, state_id) serves it.
SELECT EXISTS (SELECT 1 FROM issues WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL) AS remain;

-- name: SubscriberIDs :many
-- The accounts of the work item's undeleted subscribers' rows, in id order (M7 P1b spec 2.4, 2.6): those the fan-out of
-- an event of it notifies, the ones the write itself subscribed too. Each once: the partial unique key on (issue_id,
-- subscriber_id) holds one undeleted row of an account, and serves the read in this order. A plain read of what is
-- committed, in the fan-out's transaction after its FOR SHARE of the work item's row (ShareIssue): no lock of its own.
SELECT subscriber_id
FROM issue_subscribers
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL
ORDER BY subscriber_id;

-- name: NotificationIssue :one
-- The undeleted work item's own columns a notification keeps (M7 P1b spec 2.4, 2.5, 2.6), archived or not: its name, its
-- number, its state by id, whose name and group are the project module's, its creator, NULL when not known, and its
-- current assignees, the accounts of its undeleted assignees' rows (GetIssue's), each once by the partial unique key on
-- (issue_id, assignee_id), in id order, none for none. A plain read of what is committed, without a lock.
SELECT i.name, i.sequence_id, i.state_id, i.created_by_id,
       array(SELECT a.assignee_id FROM issue_assignees a WHERE a.issue_id = i.id AND a.deleted_at IS NULL
             ORDER BY a.assignee_id)::uuid[] AS assignee_ids
FROM issues i
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL;

-- name: CommentText :one
-- The plain text the undeleted comment stores, comment_stripped (M7 P1b spec 2.5, R6): a notification's excerpt of it.
-- A deleted comment has none. A plain read of what is committed, without a lock.
SELECT comment_stripped
FROM issue_comments
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
