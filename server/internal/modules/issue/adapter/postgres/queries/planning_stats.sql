-- name: CycleItems :many
-- The work items of the cycle cycle_id of the project that planning computes its progress from (M6 P4 spec 2.1, R1,
-- R2), in id order, each with its state, its completion and its undeleted assignees and labels, each once, in id
-- order (planning does not count one twice). It reads in the transaction ctx carries: a read's read-only snapshot, or
-- a freeze's or a transfer's write under the project's lock. A work item counts by an undeleted membership when it is
-- the project's, undeleted and not archived (a sub-item too; a draft is no work item). viewer is an OwnOnly caller,
-- who counts only the work items he created, none whose creator is gone (list_issues.sql's predicate,
-- domain.HiddenFrom); NULL counts every one. The states are the project module's: a state is answered by its id and
-- planning reads its group. cycle_issues_cycle_id_idx serves it.
SELECT i.id, i.state_id, i.completed_at,
       array(SELECT DISTINCT a.assignee_id FROM issue_assignees a WHERE a.issue_id = i.id AND a.deleted_at IS NULL
             ORDER BY a.assignee_id)::uuid[] AS assignee_ids,
       array(SELECT DISTINCT l.label_id FROM issue_labels l WHERE l.issue_id = i.id AND l.deleted_at IS NULL
             ORDER BY l.label_id)::uuid[] AS label_ids
FROM cycle_issues c
JOIN issues i ON i.id = c.issue_id
WHERE c.cycle_id = sqlc.arg(cycle_id) AND c.deleted_at IS NULL
  AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(viewer)::uuid IS NULL OR i.created_by_id = sqlc.narg(viewer)::uuid)
ORDER BY i.id;

-- name: ModuleItems :many
-- CycleItems of the module module_id's work items. module_issues_module_id_idx serves it.
SELECT i.id, i.state_id, i.completed_at,
       array(SELECT DISTINCT a.assignee_id FROM issue_assignees a WHERE a.issue_id = i.id AND a.deleted_at IS NULL
             ORDER BY a.assignee_id)::uuid[] AS assignee_ids,
       array(SELECT DISTINCT l.label_id FROM issue_labels l WHERE l.issue_id = i.id AND l.deleted_at IS NULL
             ORDER BY l.label_id)::uuid[] AS label_ids
FROM module_issues m
JOIN issues i ON i.id = m.issue_id
WHERE m.module_id = sqlc.arg(module_id) AND m.deleted_at IS NULL
  AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(viewer)::uuid IS NULL OR i.created_by_id = sqlc.narg(viewer)::uuid)
ORDER BY i.id;

-- name: CycleStateCounts :many
-- A page of cycles' work items by state, in one statement (M6 P4 spec R2, R9): of each of cycle_ids that is the
-- project's, how many of the work items CycleItems reads are in each state, by cycle and state in id order; a cycle
-- with none has no row. A work item is in one cycle at most (cycle_issues_issue_id_key), so it counts once.
SELECT c.cycle_id AS target_id, i.state_id, count(*)::bigint AS issues
FROM cycle_issues c
JOIN issues i ON i.id = c.issue_id
WHERE c.cycle_id = ANY (sqlc.arg(cycle_ids)::uuid[]) AND c.deleted_at IS NULL
  AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(viewer)::uuid IS NULL OR i.created_by_id = sqlc.narg(viewer)::uuid)
GROUP BY c.cycle_id, i.state_id
ORDER BY c.cycle_id, i.state_id;

-- name: ModuleStateCounts :many
-- CycleStateCounts of a page of modules. A work item may be in several modules, once in each
-- (module_issues_issue_id_module_id_key), and counts in each.
SELECT m.module_id AS target_id, i.state_id, count(*)::bigint AS issues
FROM module_issues m
JOIN issues i ON i.id = m.issue_id
WHERE m.module_id = ANY (sqlc.arg(module_ids)::uuid[]) AND m.deleted_at IS NULL
  AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(viewer)::uuid IS NULL OR i.created_by_id = sqlc.narg(viewer)::uuid)
GROUP BY m.module_id, i.state_id
ORDER BY m.module_id, i.state_id;

-- name: CycleAssignees :many
-- A page of cycles' assignees, in one statement (M6 P4 spec R9): of each of cycle_ids that is the project's, the
-- undeleted assignees of the work items CycleItems reads, each once, by cycle and assignee in id order.
SELECT DISTINCT c.cycle_id, a.assignee_id
FROM cycle_issues c
JOIN issues i ON i.id = c.issue_id
JOIN issue_assignees a ON a.issue_id = i.id
WHERE c.cycle_id = ANY (sqlc.arg(cycle_ids)::uuid[]) AND c.deleted_at IS NULL
  AND i.project_id = sqlc.arg(project_id) AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(viewer)::uuid IS NULL OR i.created_by_id = sqlc.narg(viewer)::uuid)
  AND a.deleted_at IS NULL
ORDER BY c.cycle_id, a.assignee_id;
