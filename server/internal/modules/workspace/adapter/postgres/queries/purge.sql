-- name: PurgeWorkspaceUserProperties :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM workspace_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM workspace_user_properties AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeWorkspaceMemberInvites :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM workspace_member_invites AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM workspace_member_invites AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeWorkspaceMembers :one
WITH candidates AS MATERIALIZED (
    SELECT candidate.id FROM workspace_members AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
), deleted AS (
    DELETE FROM workspace_members AS target WHERE target.deleted_at < sqlc.arg(cutoff) AND target.id IN (SELECT candidates.id FROM candidates) RETURNING target.id
)
SELECT (SELECT count(*)::integer FROM candidates) AS scanned,
       (SELECT count(*)::integer FROM deleted) AS deleted,
       coalesce((SELECT id FROM candidates ORDER BY id DESC LIMIT 1), sqlc.arg(after_id)::uuid)::uuid AS after_id;

-- name: PurgeWorkspacesCandidates :many
SELECT id FROM workspaces
WHERE deleted_at < sqlc.arg(cutoff) AND id > sqlc.arg(after_id)
ORDER BY id LIMIT sqlc.arg(batch_limit);

-- name: DeletePurgeWorkspaces :many
DELETE FROM workspaces
WHERE deleted_at < sqlc.arg(cutoff) AND id = ANY(sqlc.arg(ids)::uuid[])
RETURNING id;
