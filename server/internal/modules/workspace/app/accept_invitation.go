package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AcceptInvitationDeps are AcceptWorkspaceInvitation's collaborators.
type AcceptInvitationDeps struct {
	Accounts    Accounts
	Invitations InvitationAccepter
	Tx          shared.TxManager
	Clock       Clock
	MAC         InvitationMAC
}

// AcceptWorkspaceInvitation makes the caller a member by an invitation to
// his address: POST /api/v0/workspace-invitations/{invitation_id}/accept.
type AcceptWorkspaceInvitation struct {
	invitations InvitationAccepter
	tx          shared.TxManager
	clock       Clock
	responder   responder
}

// NewAcceptWorkspaceInvitation returns the use case.
func NewAcceptWorkspaceInvitation(d AcceptInvitationDeps) *AcceptWorkspaceInvitation {
	return &AcceptWorkspaceInvitation{invitations: d.Invitations, tx: d.Tx, clock: d.Clock,
		responder: responder{accounts: d.Accounts, invitations: d.Invitations, tokens: invitationTokens{mac: d.MAC}}}
}

// Execute takes responder's locks, the workspace FOR NO KEY UPDATE, the lock
// of a write of a membership (M3 design 3.6), then reads the clock and
// writes (M3 design 3.8). An invitation never changes an active
// membership: an active member's invitation is consumed, his role kept. A
// former member's row is restored with the invitation's role; P4 adds, when
// that role is a guest's, DemoteToGuest of his project memberships here, in
// the same transaction. Anyone else is inserted with it. Then the
// invitation is accepted, and deleted. The answer is the workspace with the
// caller's role as it now is.
func (u *AcceptWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	if err := u.responder.checkToken(id, token); err != nil {
		return domain.Workspace{}, err
	}
	var joined domain.Workspace
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		account, inv, err := u.responder.lock(ctx, actor, id, u.invitations.LockWorkspace)
		if err != nil {
			return err
		}
		now := u.clock.Now()
		m, member, err := u.invitations.MemberOf(ctx, inv.WorkspaceID, account.ID)
		if err != nil {
			return err
		}
		role := inv.Role
		switch {
		case member && m.IsActive:
			role = m.Role
		case member:
			err = u.invitations.RestoreMember(ctx, m.ID, inv.Role, account.ID, now)
		default:
			err = u.invitations.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: inv.WorkspaceID, MemberID: account.ID, Role: inv.Role,
				CreatedBy: account.ID, Now: now})
		}
		if err != nil {
			return err
		}
		if err := u.invitations.AcceptInvitation(ctx, inv.ID, account.ID, now); err != nil {
			return err
		}
		if joined, err = u.invitations.WorkspaceByID(ctx, inv.WorkspaceID); err != nil {
			return err
		}
		joined.Role = role
		return nil
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return joined, nil
}
