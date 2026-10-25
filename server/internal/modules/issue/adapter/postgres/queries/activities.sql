-- name: Activities :many
SELECT a.* FROM issue_activities a
WHERE a.issue_id = sqlc.arg(issue_id) AND a.deleted_at IS NULL
 AND (NOT sqlc.arg(has_after)::boolean OR (a.created_at,a.id) > (sqlc.arg(after_at)::timestamptz,sqlc.arg(after_id)::uuid))
 AND (a.field IS NULL OR a.field NOT IN ('parent','relates_to','duplicate','blocked_by','blocking') OR (
 (a.old_identifier IS NULL OR EXISTS (
  SELECT 1 FROM issues reference WHERE reference.id=a.old_identifier
   AND (reference.project_id=ANY(sqlc.arg(project_ids)::uuid[])
    OR (reference.project_id=ANY(sqlc.arg(own_only_project_ids)::uuid[]) AND reference.created_by_id=sqlc.arg(actor_id)::uuid))
 ))
 AND (a.new_identifier IS NULL OR EXISTS (
  SELECT 1 FROM issues reference WHERE reference.id=a.new_identifier
   AND (reference.project_id=ANY(sqlc.arg(project_ids)::uuid[])
    OR (reference.project_id=ANY(sqlc.arg(own_only_project_ids)::uuid[]) AND reference.created_by_id=sqlc.arg(actor_id)::uuid))
 ))
 ))
ORDER BY a.created_at ASC,a.id ASC LIMIT sqlc.arg(page_limit);
