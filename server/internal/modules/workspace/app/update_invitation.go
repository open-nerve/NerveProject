package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspaceInvitation changes an invitation's role: PATCH
// /api/v0/workspace-invitations/{invitation_id}.
type UpdateWorkspaceInvitation struct {
	invitations InvitationUpdater
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
	tokens      invitationTokens
}

// NewUpdateWorkspaceInvitation returns the use case.
func NewUpdateWorkspaceInvitation(invitations InvitationUpdater, auth shared.Authorizer, tx shared.TxManager, clock Clock,
	mac InvitationMAC) *UpdateWorkspaceInvitation {
	return &UpdateWorkspaceInvitation{invitations: invitations, auth: auth, tx: tx, clock: clock, tokens: invitationTokens{mac: mac}}
}

// Execute checks role, then in one transaction (M3 design 3.6): the
// invitation read for its workspace, the workspace FOR SHARE, the invitation
// FOR UPDATE read again, the decision on workspace_invitation.update, then,
// for a caller allowed to change it, a declined invitation is
// workspace.invitation_responded; then the change, at the time the clock
// gives under the locks. Only admins change invitations, so the role is
// never above the changer's (3.8). The answer carries the token.
func (u *UpdateWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.InvitationWithToken{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.InvitationWithToken{}, err
	}
	var updated domain.Invitation
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := readInvitation(ctx, u.invitations, id)
		if err != nil {
			return err
		}
		if inv, err = lockInvitation(ctx, u.invitations, u.invitations.ShareWorkspace, inv); err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationUpdate, inv.WorkspaceID, domain.ErrInvitationNotFound); err != nil {
			return err
		}
		if inv.Responded() {
			return domain.ErrInvitationResponded
		}
		updated, err = u.invitations.UpdateInvitationRole(ctx, inv.ID, role, actor.UserID, u.clock.Now())
		return err
	})
	if err != nil {
		return domain.InvitationWithToken{}, err
	}
	return domain.InvitationWithToken{Invitation: updated, Token: u.tokens.of(updated.ID)}, nil
}
