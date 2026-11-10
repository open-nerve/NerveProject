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

-- name: InsertModuleIssues :exec
-- A work item's new modules' rows (M6 P3a spec R4), one statement, as InsertLabels: the modules it joins. The partial
-- unique key module_issues_issue_id_module_id_key refuses a second undeleted row of a module it is in.
INSERT INTO module_issues (id, workspace_id, project_id, issue_id, module_id, created_by_id, updated_by_id, created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id),
       unnest(sqlc.arg(module_ids)::uuid[]), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);

-- name: RemoveModuleIssues :exec
-- The modules a work item leaves (M6 P3a spec R4), as RemoveLabels: their undeleted rows soft-deleted; the partial
-- unique key on (issue_id, module_id) serves it.
UPDATE module_issues
SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
WHERE issue_id = sqlc.arg(issue_id) AND module_id = ANY (sqlc.arg(module_ids)::uuid[]) AND deleted_at IS NULL;

-- name: RemoveCycleMemberships :many
-- A deleted cycle's memberships (M6 P3a spec 2.5, R6), in the deletion's transaction: its undeleted rows, locked FOR
-- NO KEY UPDATE in id order, then soft-deleted at the deletion's moment by its account, in one statement (convention
-- 5); a row deleted before, with its work item or by a leave, keeps its deletion. It answers the work items the rows
-- were of, in id order. cycle_issues_cycle_id_idx serves it.
WITH removed AS (
    UPDATE cycle_issues AS t
    SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
    WHERE t.id IN (
        SELECT r.id FROM cycle_issues r
        WHERE r.cycle_id = sqlc.arg(cycle_id) AND r.deleted_at IS NULL
        ORDER BY r.id FOR NO KEY UPDATE)
    RETURNING t.issue_id
)
SELECT issue_id FROM removed ORDER BY issue_id;

-- name: RemoveModuleMemberships :many
-- RemoveCycleMemberships' statement, of a deleted module's memberships. module_issues_module_id_idx serves it.
WITH removed AS (
    UPDATE module_issues AS t
    SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
    WHERE t.id IN (
        SELECT r.id FROM module_issues r
        WHERE r.module_id = sqlc.arg(module_id) AND r.deleted_at IS NULL
        ORDER BY r.id FOR NO KEY UPDATE)
    RETURNING t.issue_id
)
SELECT issue_id FROM removed ORDER BY issue_id;

-- name: RemoveTransferredCycleIssues :many
-- A transfer's leaving of the cycle cycle_id (M6 P4 spec 2.6, R8), in the transfer's transaction under the project's
-- row lock planning holds: of the cycle's undeleted memberships, those whose work items are the project's, undeleted,
-- not archived and in one of state_ids (planning reads the states' groups), locked FOR NO KEY UPDATE in id order, the
-- memberships alone (the work items' rows are written last, MarkChanged: the spec's lock order), then soft-deleted at
-- the transfer's moment by its account, never changed in place (P3a spec R4): InsertTransferredCycleIssues then
-- inserts the rows of the cycle they join, in the same transaction. Two statements, not one: a row's id is the
-- application's (v0 design, IDs), and how many move is known after this one. It answers the work items, in id order.
-- cycle_issues_cycle_id_idx serves it.
WITH removed AS (
    UPDATE cycle_issues AS t
    SET deleted_at = sqlc.arg(at)::timestamptz, updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(by)::uuid
    WHERE t.id IN (
        SELECT r.id FROM cycle_issues r
        JOIN issues i ON i.id = r.issue_id
        WHERE r.cycle_id = sqlc.arg(cycle_id) AND r.deleted_at IS NULL
          AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
          AND i.state_id = ANY (sqlc.arg(state_ids)::uuid[])
        ORDER BY r.id FOR NO KEY UPDATE OF r)
    RETURNING t.issue_id
)
SELECT issue_id FROM removed ORDER BY issue_id;

-- name: InsertTransferredCycleIssues :exec
-- A transfer's joining of the cycle cycle_id (M6 P4 spec 2.6, R8), in the transfer's transaction, after
-- RemoveTransferredCycleIssues: one new undeleted row of each of issue_ids, the work items that statement answered,
-- whose id is the one of ids at its place, in the project and its workspace, created and last changed at the
-- transfer's moment by its account, as InsertCycleIssues writes a work item's. Each one's row of the cycle it left is
-- deleted, so the partial unique key cycle_issues_issue_id_key holds.
INSERT INTO cycle_issues (id, workspace_id, project_id, issue_id, cycle_id, created_by_id, updated_by_id, created_at, updated_at)
SELECT unnest(sqlc.arg(ids)::uuid[]), sqlc.arg(workspace_id), sqlc.arg(project_id), unnest(sqlc.arg(issue_ids)::uuid[]),
       sqlc.arg(cycle_id), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at);
