-- name: ProjectDeletionStepRemaining :one
-- Candidate-only reads: no locks, ownership filtering, or cross-module joins.
SELECT CASE sqlc.arg(step)::text
WHEN 'draft_issues' THEN EXISTS (SELECT 1 FROM draft_issues r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issues' THEN EXISTS (SELECT 1 FROM issues r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_assignees' THEN EXISTS (SELECT 1 FROM issue_assignees r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_labels' THEN EXISTS (SELECT 1 FROM issue_labels r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_relations' THEN EXISTS (SELECT 1 FROM issue_relations r WHERE (r.project_id = sqlc.arg(project_id)::uuid OR r.related_issue_id IN (SELECT i.id FROM issues i WHERE i.project_id = sqlc.arg(project_id)))
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_links' THEN EXISTS (SELECT 1 FROM issue_links r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_subscribers' THEN EXISTS (SELECT 1 FROM issue_subscribers r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_mentions' THEN EXISTS (SELECT 1 FROM issue_mentions r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_comments' THEN EXISTS (SELECT 1 FROM issue_comments r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_reactions' THEN EXISTS (SELECT 1 FROM issue_reactions r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'comment_reactions' THEN EXISTS (SELECT 1 FROM comment_reactions r WHERE r.comment_id IN (SELECT c.id FROM issue_comments c WHERE c.project_id = sqlc.arg(project_id)::uuid)
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_activities' THEN EXISTS (SELECT 1 FROM issue_activities r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
WHEN 'issue_description_versions' THEN EXISTS (SELECT 1 FROM issue_description_versions r WHERE r.project_id = sqlc.arg(project_id)::uuid
    AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid)
ELSE false END AS remaining;

-- name: WorkspaceDraftDeletionRemaining :one
SELECT EXISTS (SELECT 1 FROM draft_issues r
 WHERE r.workspace_id = sqlc.arg(workspace_id)::uuid AND r.project_id IS NULL
 AND r.deleted_at IS NULL AND r.id > sqlc.arg(after)::uuid);
