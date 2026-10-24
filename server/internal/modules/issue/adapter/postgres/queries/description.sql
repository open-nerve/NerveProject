-- name: WriteDescription :exec
-- Persist sanitized content under the project lock. Audit columns are owned by the caller's ordinary write.
UPDATE issues
SET description_html = sqlc.arg(html), description_stripped = sqlc.arg(text)::text
WHERE id = sqlc.arg(issue_id);
