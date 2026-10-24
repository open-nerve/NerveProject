-- name: InsertCommentReaction :exec
INSERT INTO comment_reactions(id,workspace_id,project_id,comment_id,actor_id,reaction,created_by_id,updated_by_id,created_at,updated_at)
VALUES(sqlc.arg(id),sqlc.arg(workspace_id),sqlc.arg(project_id),sqlc.arg(comment_id),sqlc.arg(actor_id),sqlc.arg(reaction),sqlc.arg(by)::uuid,sqlc.arg(by),sqlc.arg(at)::timestamptz,sqlc.arg(at));

-- name: CommentReaction :one
SELECT id,workspace_id,project_id,comment_id,actor_id,reaction,created_at FROM comment_reactions
WHERE comment_id=sqlc.arg(comment_id) AND actor_id=sqlc.arg(actor_id) AND reaction=sqlc.arg(reaction) AND deleted_at IS NULL;

-- name: CommentReactions :many
SELECT id,workspace_id,project_id,comment_id,actor_id,reaction,created_at FROM comment_reactions
WHERE comment_id=ANY(sqlc.arg(comment_ids)::uuid[]) AND deleted_at IS NULL ORDER BY created_at,id;

-- name: DeleteCommentReaction :exec
UPDATE comment_reactions SET deleted_at=sqlc.arg(at)::timestamptz,updated_at=sqlc.arg(at),updated_by_id=sqlc.arg(by)::uuid
WHERE comment_id=sqlc.arg(comment_id) AND actor_id=sqlc.arg(actor_id) AND reaction=sqlc.arg(reaction) AND deleted_at IS NULL;

-- name: DeleteCommentReactionRows :exec
UPDATE comment_reactions SET deleted_at=sqlc.arg(at)::timestamptz,updated_at=sqlc.arg(at),updated_by_id=sqlc.arg(by)::uuid
WHERE comment_id=sqlc.arg(comment_id) AND deleted_at IS NULL;
