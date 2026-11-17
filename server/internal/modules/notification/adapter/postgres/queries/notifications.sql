-- name: InsertNotifications :exec
-- A fan-out's notifications (M7 P1b spec 2.9), one statement under the work item's row FOR SHARE, which the fan-out
-- took first: the n-th of ids, receiver_ids, senders, data and change_seqs is one row; every row of the work item
-- issue_id, of the workspace and the project of its locked row, sent by triggered_by_id (NULL for nerve's own write)
-- at the event's moment `at`, and written by nerve, so no audit account. A row of the same event, change and receiver
-- as an undeleted one is not inserted (the partial unique key, R4): a retried or a repeated job writes none twice, and
-- a deleted one does not keep a new one out. The foreign keys' checks take FOR KEY SHARE of the workspace's, the
-- project's and the accounts' rows, which their writes' FOR NO KEY UPDATE does not wait for, nor they for it.
INSERT INTO notifications (id, workspace_id, project_id, receiver_id, triggered_by_id, issue_id, issue_created_by_id, sender,
                           data, event_id, change_seq, created_at, updated_at)
SELECT r.id, sqlc.arg(workspace_id), sqlc.arg(project_id), r.receiver_id, sqlc.narg(triggered_by_id)::uuid, sqlc.arg(issue_id),
       sqlc.narg(issue_created_by_id)::uuid, r.sender, r.data, sqlc.arg(event_id), r.change_seq, sqlc.arg(at)::timestamptz,
       sqlc.arg(at)
FROM ROWS FROM (unnest(sqlc.arg(ids)::uuid[]), unnest(sqlc.arg(receiver_ids)::uuid[]), unnest(sqlc.arg(senders)::text[]),
                unnest(sqlc.arg(data)::jsonb[]), unnest(sqlc.arg(change_seqs)::smallint[]))
         AS r(id, receiver_id, sender, data, change_seq)
ON CONFLICT (event_id, change_seq, receiver_id) WHERE deleted_at IS NULL DO NOTHING;
