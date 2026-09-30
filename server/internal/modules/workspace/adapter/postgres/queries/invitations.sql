-- name: CreateInvitation :one
-- createWorkspaceInvitations inserts a batch one row a statement, in the order of the normalized addresses; the caller
-- holds the workspace's FOR SHARE (and the inviter's account row) (M3 design 3.6 convention 5). RETURNING gives the row
-- as stored.
INSERT INTO workspace_member_invites (id, workspace_id, email, role, created_by_id, updated_by_id, created_at, updated_at)
VALUES (sqlc.arg(id), sqlc.arg(workspace_id), sqlc.arg(email), sqlc.arg(role),
        sqlc.arg(created_by), sqlc.arg(created_by), sqlc.arg(now), sqlc.arg(now))
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: ListInvitations :many
-- listWorkspaceInvitations: the workspace's undeleted invitations, pending or declined, newest first, then by id
-- (M3 design 3.12).
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL
ORDER BY created_at DESC, id;

-- name: InvitationByID :one
-- A write on an invitation reads it before its workspace's lock, for the workspace (M3 design 3.6 convention 2).
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE id = sqlc.arg(id) AND deleted_at IS NULL;

-- name: InvitationPreview :one
-- getWorkspaceInvitation: what the link shows, never the address (M3 design 3.8). One module's tables, so one JOIN.
SELECT i.id, i.role, i.responded_at, w.name AS workspace_name, w.slug AS workspace_slug
FROM workspace_member_invites i
JOIN workspaces w ON w.id = i.workspace_id
WHERE i.id = sqlc.arg(id) AND i.deleted_at IS NULL AND w.deleted_at IS NULL;

-- name: LockInvitation :one
-- Then, under the workspace's lock, the invitation row FOR UPDATE, read again: a response, a change or a deletion that
-- committed while the write waited is seen, and none commits before it ends. After a wait, Postgres evaluates
-- deleted_at IS NULL again on the row's newest version, so an invitation deleted meanwhile reads no row.
SELECT id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at
FROM workspace_member_invites
WHERE id = sqlc.arg(id) AND deleted_at IS NULL
FOR UPDATE;

-- name: UpdateInvitationRole :one
-- updateWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE.
UPDATE workspace_member_invites
SET role = sqlc.arg(role), updated_by_id = sqlc.arg(updated_by), updated_at = sqlc.arg(now)
WHERE id = sqlc.arg(id)
RETURNING id, workspace_id, email, role, accepted, responded_at, created_by_id, updated_by_id, created_at, updated_at, deleted_at;

-- name: DeleteInvitation :exec
-- deleteWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE: the address is free again
-- at once (the partial unique index).
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE id = sqlc.arg(id);

-- name: AcceptInvitation :exec
-- acceptWorkspaceInvitation, under the workspace's FOR NO KEY UPDATE and the invitation's FOR UPDATE: accepted, and
-- deleted at the same moment (M3 design 3.8).
UPDATE workspace_member_invites
SET accepted = true, responded_at = sqlc.arg(now)::timestamptz, deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now),
    updated_by_id = sqlc.arg(accepted_by)::uuid
WHERE id = sqlc.arg(id);

-- name: DeclineInvitation :exec
-- declineWorkspaceInvitation, under the workspace's FOR SHARE and the invitation's FOR UPDATE: it stays, and holds its
-- address (M3 design 3.8).
UPDATE workspace_member_invites
SET responded_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(declined_by)::uuid
WHERE id = sqlc.arg(id);

-- name: DeleteWorkspaceInvitations :exec
-- deleteWorkspace's cascade: every undeleted invitation of the workspace, pending or declined, one statement in scan
-- order under the workspace's FOR NO KEY UPDATE (M3 design 3.6 convention 5). A row deleted before keeps its time.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE workspace_id = sqlc.arg(workspace_id) AND deleted_at IS NULL;
