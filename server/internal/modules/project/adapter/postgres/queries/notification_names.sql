-- name: NotificationState :one
-- The name and the group of a work item's state a notification's snapshot keeps (M7 P1b spec 2.5, 2.6): the
-- project's undeleted state id, its triage state too, read alone and without a lock, as the fan-out holds a work
-- item's row and takes no lock of a project; no row for none.
SELECT name, "group"
FROM states
WHERE id = sqlc.arg(id) AND project_id = sqlc.arg(project_id) AND deleted_at IS NULL;

-- name: NotificationIdentifier :one
-- The identifier of a work item's project a notification's snapshot keeps (M7 P1b spec 2.5, 2.6): the workspace's
-- undeleted project id, archived or not, read alone and without a lock, as NotificationState; no row for none.
SELECT identifier
FROM projects
WHERE id = sqlc.arg(id) AND workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
