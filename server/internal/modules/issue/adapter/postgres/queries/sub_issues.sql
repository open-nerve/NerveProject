-- name: SubIssueCounts :many
-- For each of the work items ids that has any, the number of its undeleted, unarchived direct sub-items, and of those
-- the account actor created (M4 design 3.9): a work item's sub_issues_count as its reader sees it. Read without a lock,
-- or under the project's lock of the write that answers. issues_parent_id_idx serves it.
SELECT parent_id::uuid AS parent_id, count(*)::int AS all_count,
       (count(*) FILTER (WHERE created_by_id = sqlc.arg(actor)::uuid))::int AS own_count
FROM issues
WHERE parent_id = ANY (sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL AND archived_at IS NULL
GROUP BY parent_id;
