-- name: LockMemberWorkspaces :many
-- The first step of ending a deactivated account's memberships (M3 design 3.6 convention 6, 3.9), under the account
-- row's FOR NO KEY UPDATE, which the deactivation took first: the undeleted workspaces of which he is an active member,
-- found now, locked FOR NO KEY UPDATE in id order. The lock is taken as the sorted rows come, so the order is the ids'.
-- After a wait, Postgres evaluates deleted_at IS NULL again on the row's newest version: a workspace deleted meanwhile is
-- left out, not an error, its memberships ended with it (review spike 15).
SELECT w.id
FROM workspaces w
WHERE w.deleted_at IS NULL
  AND EXISTS (SELECT 1 FROM workspace_members m
              WHERE m.workspace_id = w.id AND m.member_id = sqlc.arg(member_id) AND m.is_active AND m.deleted_at IS NULL)
ORDER BY w.id
FOR NO KEY UPDATE;

-- name: SoleAdmin :one
-- The second step, under the workspaces' locks: whether the account is the only active admin of one of the workspaces
-- that has another active member, whom ending his membership would leave without an admin (M3 design 3.7 rule 2). A
-- workspace where he is alone, or that has another active admin, does not count.
SELECT EXISTS (
    SELECT 1 FROM workspace_members m
    WHERE m.workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND m.member_id = sqlc.arg(member_id) AND m.role = 20
      AND m.is_active AND m.deleted_at IS NULL
      AND NOT EXISTS (SELECT 1 FROM workspace_members a
                      WHERE a.workspace_id = m.workspace_id AND a.member_id <> m.member_id AND a.role = 20 AND a.is_active
                        AND a.deleted_at IS NULL)
      AND EXISTS (SELECT 1 FROM workspace_members o
                  WHERE o.workspace_id = m.workspace_id AND o.member_id <> m.member_id AND o.is_active
                    AND o.deleted_at IS NULL));

-- name: DeleteInvitationsTo :exec
-- The third step (M3 design 3.8, 3.9; Plane views/user/base.py:313): every undeleted invitation to the address, of
-- every workspace, pending or declined, soft-deleted at the moment and by the account given, before the memberships'
-- rows (the global order). One statement in scan order; a deleted one keeps its moment.
UPDATE workspace_member_invites
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE email = sqlc.arg(email) AND deleted_at IS NULL;

-- name: DeleteInvitationsOfWorkspacesLeftEmpty :exec
-- The fourth step, under the workspaces' locks (M3 design 3.7, 3.9; the P6 pre-flight's M1): the pending invitations of
-- each of the workspaces where the account has no other active member, soft-deleted at the moment and by the account
-- given, before the memberships' rows (the global order). Run before his memberships end, "other" leaves him out. A
-- workspace he was alone in then has no active member, and no invitation lets anyone into it: one accepted would make
-- a member of a workspace with no admin. A declined one stays, which cannot be accepted; a deleted one keeps its
-- moment. Creating an invitation holds the workspace FOR SHARE, accepting one FOR NO KEY UPDATE: neither commits while
-- the deactivation holds it.
UPDATE workspace_member_invites i
SET deleted_at = sqlc.arg(now)::timestamptz, updated_at = sqlc.arg(now), updated_by_id = sqlc.arg(deleted_by)::uuid
WHERE i.workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND i.responded_at IS NULL AND i.deleted_at IS NULL
  AND NOT EXISTS (SELECT 1 FROM workspace_members o
                  WHERE o.workspace_id = i.workspace_id AND o.member_id <> sqlc.arg(member_id) AND o.is_active
                    AND o.deleted_at IS NULL);

-- name: EndWorkspaceMemberships :exec
-- The fifth step, one statement under the workspaces' locks (convention 5): the account's active memberships of the
-- workspaces end, at the moment and by the account given; the rows stay, each with its role, and an ended or deleted one
-- keeps its columns.
UPDATE workspace_members
SET is_active = false, updated_at = sqlc.arg(now)::timestamptz, updated_by_id = sqlc.arg(ended_by)::uuid
WHERE workspace_id = ANY (sqlc.arg(workspace_ids)::uuid[]) AND member_id = sqlc.arg(member_id) AND is_active
  AND deleted_at IS NULL;
