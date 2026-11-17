-- name: ProjectIssuesRemain :one
-- Whether the project has an undeleted work item, archived or not (M7 P1b spec 2.6, 2.8, R3): the deletion of a deleted
-- project's notifications waits until issue.delete_rows has soft-deleted every one of them, so no fan-out, which holds
-- its work item's row FOR SHARE and reads it undeleted, can write a notification of the project after it. A plain read
-- of what is committed: no lock. The partial index on (project_id, state_id) serves it.
SELECT EXISTS (SELECT 1 FROM issues WHERE project_id = sqlc.arg(project_id) AND deleted_at IS NULL) AS remain;
