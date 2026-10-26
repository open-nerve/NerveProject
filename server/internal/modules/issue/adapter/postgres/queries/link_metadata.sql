-- name: SetLinkMetadata :execrows
UPDATE issue_links SET metadata = sqlc.arg(metadata)
WHERE id = sqlc.arg(id) AND url = sqlc.arg(url) AND deleted_at IS NULL;
