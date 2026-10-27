-- name: PurgeWorkspaceUserProperties :many
DELETE FROM workspace_user_properties
WHERE id IN (
    SELECT candidate.id FROM workspace_user_properties AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeWorkspaceMemberInvites :many
DELETE FROM workspace_member_invites
WHERE id IN (
    SELECT candidate.id FROM workspace_member_invites AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeWorkspaceMembers :many
DELETE FROM workspace_members
WHERE id IN (
    SELECT candidate.id FROM workspace_members AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

-- name: PurgeWorkspaces :many
DELETE FROM workspaces
WHERE id IN (
    SELECT candidate.id FROM workspaces AS candidate
    WHERE candidate.deleted_at < sqlc.arg(cutoff) AND candidate.id > sqlc.arg(after_id)
    ORDER BY candidate.id LIMIT sqlc.arg(batch_limit)
)
RETURNING id;

