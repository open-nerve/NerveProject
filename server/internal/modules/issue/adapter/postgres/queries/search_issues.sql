-- name: SearchIssues :many
SELECT id, similarity(name, @query) AS similarity
FROM issues
WHERE deleted_at IS NULL;
