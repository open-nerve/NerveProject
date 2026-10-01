package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AcceptInvitationDeps are AcceptWorkspaceInvitation's collaborators.
type AcceptInvitationDeps struct {
	Accounts    Accounts
	Invitations InvitationAccepter
	Projects    ProjectCascade
	Tx          shared.TxManager
	Clock       Clock
	MAC         InvitationMAC
}

// AcceptWorkspaceInvitation makes the caller a member by an invitation to
// his address: POST /api/v0/workspace-invitations/{invitation_id}/accept.
type AcceptWorkspaceInvitation struct {
	invitations InvitationAccepter
	projects    ProjectCascade
	tx          shared.TxManager
	clock       Clock
	responder   responder
}

// NewAcceptWorkspaceInvitation returns the use case.
func NewAcceptWorkspaceInvitation(d AcceptInvitationDeps) *AcceptWorkspaceInvitation {
	return &AcceptWorkspaceInvitation{invitations: d.Invitations, projects: d.Projects, tx: d.Tx, clock: d.Clock,
		responder: responder{accounts: d.Accounts, invitations: d.Invitations, tokens: invitationTokens{mac: d.MAC}}}
}

// Execute takes responder's locks, the workspace FOR NO KEY UPDATE, the lock
// of a write of a membership (M3 design 3.6), then reads the clock and
// writes (M3 design 3.8). An invitation never changes an active
// membership: an active member's invitation is consumed, his role kept. A
// former member's row is restored with the invitation's role (restore).
// Anyone else is inserted with it. Then the invitation is accepted, and
// deleted. The answer is the workspace with the caller's role as it now
// is.
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
			err = u.restore(ctx, m, inv.Role, now)
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

// restore makes the ended membership m active again with role, by its
// member at now. Restored as a guest, he is made a guest in each of the
// workspace's projects he has a membership of, ended ones too
// (ProjectCascade.DemoteToGuest, M3 design 3.8), in the same transaction:
// joining a project again restores his membership there at no more than
// its role before, so without it a former guest made a member later would
// get back the project roles he had before he was a guest.
func (u *AcceptWorkspaceInvitation) restore(ctx context.Context, m domain.Membership, role shared.Role, now time.Time) error {
	if err := u.invitations.RestoreMember(ctx, m.ID, role, m.MemberID, now); err != nil {
		return err
	}
	if role != shared.RoleGuest {
		return nil
	}
	return u.projects.DemoteToGuest(ctx, m.WorkspaceID, m.MemberID, m.MemberID, now)
}
