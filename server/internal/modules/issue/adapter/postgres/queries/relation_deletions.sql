-- name: LockProjectRelationRows :many
-- Both endpoint issues remain readable after their soft deletion. Lock only
-- relations, ascending, before reading project deletion facts through the port.
SELECT r.id, i.project_id, j.project_id AS related_project_id
FROM issue_relations r
JOIN issues i ON i.id = r.issue_id
JOIN issues j ON j.id = r.related_issue_id
WHERE (i.project_id = sqlc.arg(project_id) OR j.project_id = sqlc.arg(project_id))
  AND r.deleted_at IS NULL AND r.id > sqlc.arg(after_id)
ORDER BY r.id LIMIT sqlc.arg(batch_limit) FOR NO KEY UPDATE OF r;

-- name: LockIssueRelationRows :many
SELECT r.id, i.project_id, j.project_id AS related_project_id
FROM issue_relations r
JOIN issues i ON i.id = r.issue_id
JOIN issues j ON j.id = r.related_issue_id
WHERE (r.issue_id = ANY(sqlc.arg(issue_ids)::uuid[]) OR r.related_issue_id = ANY(sqlc.arg(issue_ids)::uuid[]))
  AND r.deleted_at IS NULL
ORDER BY r.id FOR NO KEY UPDATE OF r;

-- name: DeleteLockedRelationRows :execrows
-- The caller supplies only IDs it locked and owns in this same transaction.
UPDATE issue_relations SET deleted_at = sqlc.arg(at)::timestamptz,
  updated_at = sqlc.arg(at), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = ANY(sqlc.arg(ids)::uuid[]) AND deleted_at IS NULL;
