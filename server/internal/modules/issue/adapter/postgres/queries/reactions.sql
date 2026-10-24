-- name: InsertIssueReaction :exec
INSERT INTO issue_reactions(id,workspace_id,project_id,issue_id,actor_id,reaction,created_by_id,updated_by_id,created_at,updated_at)
VALUES(sqlc.arg(id),sqlc.arg(workspace_id),sqlc.arg(project_id),sqlc.arg(issue_id),sqlc.arg(actor_id),sqlc.arg(reaction),sqlc.arg(by)::uuid,sqlc.arg(by),sqlc.arg(at)::timestamptz,sqlc.arg(at));

-- name: IssueReaction :one
SELECT id,workspace_id,project_id,issue_id,actor_id,reaction,created_at FROM issue_reactions
WHERE issue_id=sqlc.arg(issue_id) AND actor_id=sqlc.arg(actor_id) AND reaction=sqlc.arg(reaction) AND deleted_at IS NULL;

-- name: IssueReactions :many
SELECT id,workspace_id,project_id,issue_id,actor_id,reaction,created_at FROM issue_reactions
WHERE issue_id=sqlc.arg(issue_id) AND deleted_at IS NULL ORDER BY created_at,id;

-- name: DeleteIssueReaction :exec
UPDATE issue_reactions SET deleted_at=sqlc.arg(at)::timestamptz,updated_at=sqlc.arg(at),updated_by_id=sqlc.arg(by)::uuid
WHERE issue_id=sqlc.arg(issue_id) AND actor_id=sqlc.arg(actor_id) AND reaction=sqlc.arg(reaction) AND deleted_at IS NULL;
