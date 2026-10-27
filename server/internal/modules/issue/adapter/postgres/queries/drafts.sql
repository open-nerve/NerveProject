-- name: GetDraft :one
SELECT id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at
FROM draft_issues WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: LockDraft :one
SELECT id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at
FROM draft_issues WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR UPDATE;

-- name: ListDraftRows :many
SELECT id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at
FROM draft_issues
WHERE workspace_id = sqlc.arg(workspace_id) AND created_by_id = sqlc.arg(actor_id) AND deleted_at IS NULL
    AND (project_id IS NULL OR project_id = ANY(sqlc.arg(project_ids)::uuid[]))
    AND (NOT sqlc.arg(has_after)::boolean OR (created_at, id) < (sqlc.arg(after_at)::timestamptz, sqlc.arg(after_id)::uuid))
ORDER BY created_at DESC, id DESC LIMIT sqlc.arg(page_limit);

-- name: CountDraftRows :one
SELECT count(*) FROM draft_issues
WHERE workspace_id = sqlc.arg(workspace_id) AND created_by_id = sqlc.arg(actor_id) AND deleted_at IS NULL
    AND (project_id IS NULL OR project_id = ANY(sqlc.arg(project_ids)::uuid[]));

-- name: InsertDraft :one
INSERT INTO draft_issues (id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.narg(project_id), sqlc.arg(payload),
    sqlc.arg(by)::uuid, sqlc.arg(by), sqlc.arg(at)::timestamptz, sqlc.arg(at))
RETURNING id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at;

-- name: UpdateDraft :one
WITH changed AS (
    UPDATE draft_issues AS d SET project_id = sqlc.narg(project_id)::uuid, payload = sqlc.arg(payload)::jsonb,
        updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)::timestamptz
    WHERE d.id = sqlc.arg(id) AND d.deleted_at IS NULL
        AND (d.project_id IS DISTINCT FROM sqlc.narg(project_id)::uuid OR d.payload IS DISTINCT FROM sqlc.arg(payload)::jsonb)
    RETURNING d.id, d.workspace_id, d.project_id, d.payload, d.created_by_id, d.updated_by_id, d.created_at, d.updated_at
)
SELECT id, workspace_id, project_id, payload, created_by_id, updated_by_id, created_at, updated_at FROM changed
UNION ALL
SELECT d.id, d.workspace_id, d.project_id, d.payload, d.created_by_id, d.updated_by_id, d.created_at, d.updated_at
FROM draft_issues AS d WHERE d.id = sqlc.arg(id) AND d.deleted_at IS NULL AND NOT EXISTS (SELECT 1 FROM changed);

-- name: DeleteDraft :exec
UPDATE draft_issues SET deleted_at = sqlc.arg(at)::timestamptz,
    updated_by_id = sqlc.arg(by)::uuid, updated_at = sqlc.arg(at)
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;
