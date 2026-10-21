-- name: SubIssueCounts :many
-- For each of the work items ids that has any, the number of its undeleted, unarchived direct sub-items, and of those
-- the account actor created (M4 design 3.9): a work item's sub_issues_count as its reader sees it. Read without a lock,
-- or under the project's lock of the write that answers. issues_parent_id_idx serves it.
SELECT parent_id::uuid AS parent_id, count(*)::int AS all_count,
       (count(*) FILTER (WHERE created_by_id = sqlc.arg(actor)::uuid))::int AS own_count
FROM issues
WHERE parent_id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL AND archived_at IS NULL
GROUP BY parent_id;

-- name: SubIssues :many
-- The undeleted, unarchived work items directly under the work item parent_id, each as GetIssue reads it, those the
-- account created_by created when it is given, in the order of their creation (M4 design 3.4, 3.9): listSubIssues',
-- read without a lock. issues_parent_id_idx serves it.
SELECT i.id, i.workspace_id, i.project_id, i.sequence_id, i.name, i.state_id, i.priority, i.start_date, i.target_date,
       i.parent_id, i.sort_order, i.completed_at, i.archived_at, i.created_by_id, i.updated_by_id, i.created_at, i.updated_at,
       (SELECT array_agg(a.assignee_id ORDER BY a.assignee_id) FROM issue_assignees a
        WHERE a.issue_id = i.id AND a.deleted_at IS NULL)::uuid[] AS assignee_ids,
       (SELECT array_agg(l.label_id ORDER BY l.label_id) FROM issue_labels l
        WHERE l.issue_id = i.id AND l.deleted_at IS NULL)::uuid[] AS label_ids
FROM issues i
WHERE i.parent_id = sqlc.arg(parent_id)::uuid AND i.deleted_at IS NULL AND i.archived_at IS NULL
  AND (sqlc.narg(created_by)::uuid IS NULL OR i.created_by_id = sqlc.narg(created_by)::uuid)
ORDER BY i.created_at, i.id;

-- name: SetParents :execrows
-- addSubIssues' work items put under their parent (M4 design 3.9), each last changed at the moment and by the account
-- given; the undeleted ones among ids. Under the project's FOR NO KEY UPDATE, which every write of the project's work
-- items holds (3.5): one statement in scan order (convention 5).
UPDATE issues
SET parent_id = sqlc.arg(parent_id)::uuid, updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;
