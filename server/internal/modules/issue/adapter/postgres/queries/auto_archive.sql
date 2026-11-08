-- name: AutoArchive :many
-- The caller holds the project's lock and supplies its terminal state IDs.
-- An empty state array matches nothing; no other module's table is read.
-- A work item in one of the open cycles or modules the caller supplies
-- (planning's, read under that lock), by an undeleted row of its own, is
-- kept (M6 P3a spec R9); an empty or null array keeps none. The partial
-- unique keys on cycle_issues (issue_id) and module_issues (issue_id,
-- module_id), both WHERE deleted_at IS NULL, serve the two probes.
UPDATE issues
SET archived_at = sqlc.arg(at)::timestamptz,
    updated_at = sqlc.arg(at)::timestamptz, updated_by_id = NULL
WHERE issues.project_id = sqlc.arg(project_id)
  AND issues.deleted_at IS NULL AND issues.archived_at IS NULL
  AND issues.state_id = ANY (sqlc.arg(state_ids)::uuid[])
  AND issues.updated_at <= sqlc.arg(cutoff)::timestamptz
  AND NOT EXISTS (
    SELECT 1 FROM cycle_issues ci
    WHERE ci.issue_id = issues.id AND ci.deleted_at IS NULL
      AND ci.cycle_id = ANY (sqlc.arg(open_cycles)::uuid[]))
  AND NOT EXISTS (
    SELECT 1 FROM module_issues mi
    WHERE mi.issue_id = issues.id AND mi.deleted_at IS NULL
      AND mi.module_id = ANY (sqlc.arg(open_modules)::uuid[]))
RETURNING id;
