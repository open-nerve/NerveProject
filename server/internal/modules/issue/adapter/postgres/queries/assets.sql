-- name: LockAssetDrafts :many
-- Include tombstones; authorization rechecks the live draft under this lock.
SELECT id FROM draft_issues WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id FOR UPDATE;

-- name: AssetIssueFacts :many
SELECT id,deleted_at,updated_by_id AS deleted_by_id FROM issues WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id;

-- name: AssetCommentFacts :many
SELECT id,deleted_at,updated_by_id AS deleted_by_id FROM issue_comments WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id;

-- name: AssetDraftFacts :many
SELECT id,deleted_at,updated_by_id AS deleted_by_id FROM draft_issues WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id;
