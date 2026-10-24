-- name: InsertComment :exec
INSERT INTO issue_comments (id, workspace_id, project_id, issue_id, actor_id, comment_html, comment_stripped,
    created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(project_id), sqlc.arg(issue_id), sqlc.arg(actor_id),
    sqlc.arg(comment_html), sqlc.arg(comment_stripped), sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at));

-- name: Comments :many
SELECT id, workspace_id, project_id, issue_id, actor_id, comment_html, comment_stripped, edited_at, created_at, updated_at
FROM issue_comments
WHERE issue_id = sqlc.arg(issue_id) AND deleted_at IS NULL
    AND (NOT sqlc.arg(has_after)::boolean OR (created_at, id) > (sqlc.arg(after_at)::timestamptz, sqlc.arg(after_id)::uuid))
ORDER BY created_at, id LIMIT sqlc.arg(page_limit);

-- name: GetComment :one
SELECT id, workspace_id, project_id, issue_id, actor_id, comment_html, comment_stripped, edited_at, created_at, updated_at
FROM issue_comments WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: UpdateComment :exec
UPDATE issue_comments SET comment_html=sqlc.arg(comment_html), comment_stripped=sqlc.arg(comment_stripped),
    edited_at=sqlc.arg(at)::timestamptz, updated_at=sqlc.arg(at), updated_by_id=sqlc.arg(by)::uuid
WHERE id=sqlc.arg(id) AND deleted_at IS NULL;

-- name: DeleteComment :exec
UPDATE issue_comments SET deleted_at=sqlc.arg(at)::timestamptz, updated_at=sqlc.arg(at), updated_by_id=sqlc.arg(by)::uuid
WHERE id=sqlc.arg(id) AND deleted_at IS NULL;
