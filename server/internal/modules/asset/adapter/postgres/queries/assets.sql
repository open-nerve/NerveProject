-- name: FindAsset :one
SELECT * FROM file_assets WHERE id=$1;

-- name: LockAssets :many
SELECT * FROM file_assets WHERE id=ANY(sqlc.arg(ids)::uuid[]) ORDER BY id FOR UPDATE;

-- name: CreateAsset :one
INSERT INTO file_assets(id,user_id,workspace_id,project_id,issue_id,comment_id,draft_issue_id,entity_type,scope_kind,
 attributes,size,is_uploaded,is_deleted,bound_at,upload_expires_at,storage_metadata,storage_revision,
 created_by_id,updated_by_id,deleted_by_id,created_at,updated_at,deleted_at)
VALUES(sqlc.arg(id),sqlc.arg(user_id),sqlc.narg(workspace_id),sqlc.narg(project_id),sqlc.narg(issue_id),sqlc.narg(comment_id),sqlc.narg(draft_issue_id),sqlc.arg(entity_type),sqlc.arg(scope_kind),
 sqlc.arg(attributes),sqlc.arg(size),sqlc.arg(is_uploaded),sqlc.arg(is_deleted),sqlc.narg(bound_at),sqlc.narg(upload_expires_at),sqlc.arg(storage_metadata),sqlc.arg(storage_revision),
 sqlc.narg(created_by_id),sqlc.narg(updated_by_id),sqlc.narg(deleted_by_id),sqlc.arg(created_at),sqlc.arg(updated_at),sqlc.narg(deleted_at)) RETURNING *;

-- name: SaveAsset :one
UPDATE file_assets SET attributes=sqlc.arg(attributes),is_uploaded=sqlc.arg(is_uploaded),is_deleted=sqlc.arg(is_deleted),
 storage_metadata=sqlc.arg(storage_metadata),storage_revision=storage_revision+1,
 updated_by_id=sqlc.narg(updated_by_id),updated_at=sqlc.arg(updated_at),deleted_by_id=sqlc.narg(deleted_by_id),deleted_at=sqlc.narg(deleted_at)
WHERE id=sqlc.arg(id) AND storage_revision=sqlc.arg(expected_revision) RETURNING *;

-- name: ListRecovery :many
SELECT * FROM file_assets WHERE id>sqlc.arg(after_id) ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: ListScope :many
SELECT * FROM file_assets WHERE id>sqlc.arg(after_id) AND (
 (sqlc.arg(scope_kind)::text='workspace' AND workspace_id=sqlc.arg(scope_id)::uuid) OR
 (sqlc.arg(scope_kind)::text='project' AND project_id=sqlc.arg(scope_id)::uuid))
ORDER BY id LIMIT sqlc.arg(batch_limit);
