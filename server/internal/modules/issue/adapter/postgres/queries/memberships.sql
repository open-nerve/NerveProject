-- name: InsertCycleIssues :exec
-- A work item's new cycle's row (M6 P3a spec R4), one statement, as InsertLabels: the cycle it joins. A work item is
-- in one cycle at most: the partial unique key cycle_issues_issue_id_key refuses a second undeleted row.
INSERT INTO cycle_issues (id, workspace_id, project_id, issue_id, cycle_id, created_by_id, updated_by_id, created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(cycle_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: RemoveCycleIssues :exec
-- The cycle a work item leaves (M6 P3a spec R4), as RemoveLabels: its undeleted row soft-deleted, never changed in
-- place; the partial unique key on issue_id serves it.
UPDATE cycle_issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND cycle_id = ANY (sqlc.arg(cycle_ids)::uuid[]) AND deleted_at IS NULL;
