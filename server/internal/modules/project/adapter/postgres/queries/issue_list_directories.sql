-- name: IssueListStates :many
SELECT id, name, "group", "default"
FROM states
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[])
  AND deleted_at IS NULL
  AND "group" <> 'triage'
ORDER BY id;

-- name: IssueListLabels :many
SELECT id, name
FROM labels
WHERE project_id = ANY (sqlc.arg(project_ids)::uuid[])
  AND deleted_at IS NULL
ORDER BY id;
