-- name: DeleteIssueAssets :exec
WITH targets AS MATERIALIZED (
 SELECT candidate.id FROM file_assets AS candidate WHERE candidate.issue_id=sqlc.arg(parent_id) AND (candidate.deleted_at IS NULL OR candidate.deleted_at>sqlc.arg(at)) ORDER BY candidate.id FOR UPDATE
)
UPDATE file_assets AS target SET deleted_at=sqlc.arg(at),deleted_by_id=sqlc.narg(by),is_deleted=true,
 updated_at=sqlc.arg(at),updated_by_id=sqlc.narg(by),storage_revision=target.storage_revision+1
WHERE target.id IN (SELECT targets.id FROM targets) AND (target.deleted_at IS NULL OR target.deleted_at>sqlc.arg(at));

-- name: DeleteCommentAssets :exec
WITH targets AS MATERIALIZED (
 SELECT candidate.id FROM file_assets AS candidate WHERE candidate.comment_id=sqlc.arg(parent_id) AND (candidate.deleted_at IS NULL OR candidate.deleted_at>sqlc.arg(at)) ORDER BY candidate.id FOR UPDATE
)
UPDATE file_assets AS target SET deleted_at=sqlc.arg(at),deleted_by_id=sqlc.narg(by),is_deleted=true,
 updated_at=sqlc.arg(at),updated_by_id=sqlc.narg(by),storage_revision=target.storage_revision+1
WHERE target.id IN (SELECT targets.id FROM targets) AND (target.deleted_at IS NULL OR target.deleted_at>sqlc.arg(at));

-- name: DeleteDraftAssets :exec
WITH targets AS MATERIALIZED (
 SELECT candidate.id FROM file_assets AS candidate WHERE candidate.draft_issue_id=sqlc.arg(parent_id) AND (candidate.deleted_at IS NULL OR candidate.deleted_at>sqlc.arg(at)) ORDER BY candidate.id FOR UPDATE
)
UPDATE file_assets AS target SET deleted_at=sqlc.arg(at),deleted_by_id=sqlc.narg(by),is_deleted=true,
 updated_at=sqlc.arg(at),updated_by_id=sqlc.narg(by),storage_revision=target.storage_revision+1
WHERE target.id IN (SELECT targets.id FROM targets) AND (target.deleted_at IS NULL OR target.deleted_at>sqlc.arg(at));
