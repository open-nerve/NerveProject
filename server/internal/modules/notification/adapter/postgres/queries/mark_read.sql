-- name: MarkNotificationsRead :many
-- A batch of mark-all-read (M7 P1c spec 2.7, R5): read at now by the receiver, the at most `batch` of his
-- notifications after the id `after`, in id order, that the locked subquery selects: undeleted and unread, of an
-- event at or before now (the bound is the event's moment, created_at, which the request reads once; a fan-out late
-- for an earlier event may write a row a later batch marks, pre-flight re-84 I2), of the projects he reads, own-only
-- as the list (R2), of the senders, archived or not and snoozed at now or not as the filter says. Every condition is
-- in the locked subquery, which rechecks a row changed since the statement's snapshot as its lock reads it.
-- Mark-all-read never waits for a row lock: SKIP LOCKED passes over a row another transaction holds, which is only
-- one being soft-deleted, that needs no mark, or one a single change or another mark-all-read holds, for
-- milliseconds, which it does not promise to mark. Waiting would deadlock with a work item's deletion, which locks
-- its subtree's notifications work item by work item, not in one id order (re-84 I1). A row passed over takes no
-- place in the LIMIT, so a batch short of it is the last. The locked rows are a MATERIALIZED CTE, read once: an IN
-- subquery the plan rescans for each row of the table locks again on each scan, passing over the rows this
-- statement updated, so it marks past its LIMIT (T5's probe). RETURNING has no order: the caller's next `after` is
-- the greatest id it returns. The unread index (receiver_id, workspace_id) serves it but for archived ones; the id
-- order is a sort of the rows left (re-84 m7).
WITH locked AS MATERIALIZED (
    SELECT r.id FROM notifications r
    WHERE r.receiver_id = sqlc.arg(receiver_id)::uuid AND r.workspace_id = sqlc.arg(workspace_id) AND r.deleted_at IS NULL AND r.read_at IS NULL
      AND r.created_at <= sqlc.arg(now)::timestamptz
      AND EXISTS (SELECT 1 FROM ROWS FROM (unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS g(project_id, own_only)
                  WHERE g.project_id = r.project_id AND (NOT g.own_only OR r.issue_created_by_id = sqlc.arg(receiver_id)::uuid))
      AND r.sender = ANY(sqlc.arg(senders)::text[])
      AND (r.archived_at IS NOT NULL) = sqlc.arg(archived)::boolean
      AND coalesce(r.snoozed_till > sqlc.arg(now), false) = sqlc.arg(snoozed)::boolean
      AND r.id > sqlc.arg(after)::uuid
    ORDER BY r.id LIMIT sqlc.arg(batch) FOR NO KEY UPDATE SKIP LOCKED
)
UPDATE notifications AS t
SET read_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(receiver_id)::uuid
FROM locked
WHERE t.id = locked.id
RETURNING t.id;
