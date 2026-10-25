-- name: ListAssigneeIDs :many
SELECT DISTINCT a.assignee_id
FROM issue_assignees a
WHERE a.project_id = ANY (sqlc.arg(project_ids)::uuid[])
  AND a.deleted_at IS NULL
ORDER BY a.assignee_id;
