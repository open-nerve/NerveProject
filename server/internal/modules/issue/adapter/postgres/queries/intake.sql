-- name: InsertIntakeIssue :exec
-- submitIntakeIssue's intake row (M7 P2a spec 2.4), in the transaction that creates its work item, under the project's
-- FOR NO KEY UPDATE: pending, without a time or a target, created and last changed at the moment and by the submitter,
-- who created the work item too. Every write of an intake row holds that lock first (the submission and the triage,
-- the work item's deletion), and a project's or a workspace's deletion writes them in batches once the project's row is
-- deleted, so no two writes of these rows lock in opposite orders and a row takes no lock of its own (pre-flight m3).
INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id), 'pending', sqlc.arg(by)::uuid,
        sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: TriageIntakeIssue :execrows
-- triageIntakeIssue's write of the work item's undeleted intake row (M7 P2a spec 2.5), under the project's FOR NO KEY
-- UPDATE, as InsertIntakeIssue: its status, its time and its target, NULL unless the status has them (the table's
-- CHECKs), last changed at the moment and by the account given. intake_issues_issue_id_key serves it: one row at most.
UPDATE intake_issues
SET status = sqlc.arg(status), snoozed_till = sqlc.narg(snoozed_till), duplicate_to_id = sqlc.narg(duplicate_to_id),
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL;
