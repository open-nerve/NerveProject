-- name: ListNotifications :many
-- A page of a receiver's notifications in a workspace (M7 P1c spec 2.3, 2.5), newest first and then by id, after the
-- cursor's row when there is one: his undeleted ones of the projects he reads, which come as the paired arrays of
-- project ids and own-only flags, in an own-only project only those of the work items he created, so one of an
-- unknown creator is not selected (R2: the guest rule's SQL copy, here, in the unread counts and in mark-all-read's
-- batch alone, which the six-cell test holds to issue's HiddenFrom); of one of the senders; read, unread or both;
-- archived or not; snoozed at now or not, snoozed_till after now being snoozed (R6; the statement reads no clock).
-- Every condition precedes the LIMIT, which includes the caller's lookahead. The list index (receiver_id,
-- workspace_id, created_at DESC, id DESC) WHERE deleted_at IS NULL is walked in its order, each row's project looked
-- up in the arrays, so nothing is sorted. The row comparison casts both parameters, or sqlc types the id as a time.
SELECT n.id, n.workspace_id, n.project_id, n.issue_id, n.sender, n.triggered_by_id, n.data, n.read_at, n.archived_at, n.snoozed_till,
       n.created_at
FROM notifications n
WHERE n.receiver_id = sqlc.arg(receiver_id)::uuid AND n.workspace_id = sqlc.arg(workspace_id) AND n.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM ROWS FROM (unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS g(project_id, own_only)
              WHERE g.project_id = n.project_id AND (NOT g.own_only OR n.issue_created_by_id = sqlc.arg(receiver_id)::uuid))
  AND n.sender = ANY(sqlc.arg(senders)::text[])
  AND (sqlc.narg(read)::boolean IS NULL OR (n.read_at IS NOT NULL) = sqlc.narg(read)::boolean)
  AND (n.archived_at IS NOT NULL) = sqlc.arg(archived)::boolean
  AND coalesce(n.snoozed_till > sqlc.arg(now)::timestamptz, false) = sqlc.arg(snoozed)::boolean
  AND (NOT sqlc.arg(has_after)::boolean OR (n.created_at, n.id) < (sqlc.arg(after_at)::timestamptz, sqlc.arg(after_id)::uuid))
ORDER BY n.created_at DESC, n.id DESC
LIMIT sqlc.arg(page_limit);

-- name: CountUnreadNotifications :one
-- A receiver's two unread counts in a workspace (spec 2.2, R3), one for each tab, each of the senders it is given:
-- his undeleted, unread, unarchived notifications not snoozed at now, snoozed_till at or before now being over (R6;
-- the statement reads no clock), of the projects he reads, own-only as the list (R2). The unread index (receiver_id,
-- workspace_id) WHERE deleted_at IS NULL AND read_at IS NULL AND archived_at IS NULL holds the rows it counts.
SELECT count(*) FILTER (WHERE u.sender = ANY(sqlc.arg(all_senders)::text[])) AS all_unread,
       count(*) FILTER (WHERE u.sender = ANY(sqlc.arg(mention_senders)::text[])) AS mentions_unread
FROM notifications u
WHERE u.receiver_id = sqlc.arg(receiver_id)::uuid AND u.workspace_id = sqlc.arg(workspace_id) AND u.deleted_at IS NULL AND u.read_at IS NULL
  AND u.archived_at IS NULL AND (u.snoozed_till IS NULL OR u.snoozed_till <= sqlc.arg(now)::timestamptz)
  AND EXISTS (SELECT 1 FROM ROWS FROM (unnest(sqlc.arg(project_ids)::uuid[]), unnest(sqlc.arg(project_own)::boolean[])) AS g(project_id, own_only)
              WHERE g.project_id = u.project_id AND (NOT g.own_only OR u.issue_created_by_id = sqlc.arg(receiver_id)::uuid));
