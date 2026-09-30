-- name: CreateInvitation :one
-- createWorkspaceInvitations inserts a batch one row a statement, in the order of the normalized addresses, under the
-- workspace's FOR SHARE (M3 design 3.6 convention 5). RETURNING gives the row as stored.
INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(email), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: DeleteWorkspaceInvitations :exec
-- deleteWorkspace's cascade: every undeleted invitation of the workspace, pending or declined, one statement in scan
-- order under the workspace's FOR NO KEY UPDATE (M3 design 3.6 convention 5). A row deleted before keeps its time.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
