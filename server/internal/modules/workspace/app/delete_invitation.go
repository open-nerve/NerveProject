package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteWorkspaceInvitation deletes an invitation: DELETE
// /api/v0/workspace-invitations/{invitation_id}.
type DeleteWorkspaceInvitation struct {
	invitations InvitationDeleter
	auth        shared.Authorizer
	tx          shared.TxManager
	clock       Clock
}

// NewDeleteWorkspaceInvitation returns the use case.
func NewDeleteWorkspaceInvitation(invitations InvitationDeleter, auth shared.Authorizer, tx shared.TxManager, clock Clock) *DeleteWorkspaceInvitation {
	return &DeleteWorkspaceInvitation{invitations: invitations, auth: auth, tx: tx, clock: clock}
}

// Execute deletes the invitation in one transaction (M3 design 3.6): read
// for its workspace, the workspace FOR SHARE, the invitation FOR UPDATE read
// again, the decision on workspace_invitation.delete, then the soft
// deletion, at the time the clock gives under the locks. A declined
// invitation is deleted as a pending one is, and its address is free again
// (3.8); deleting and inviting again is how a leaked link is voided.
func (u *DeleteWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		inv, err := readInvitation(ctx, u.invitations, id)
		if err != nil {
			return err
		}
		if inv, err = lockInvitation(ctx, u.invitations, u.invitations.ShareWorkspace, inv); err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionInvitationDelete, inv.WorkspaceID, domain.ErrInvitationNotFound); err != nil {
			return err
		}
		return u.invitations.DeleteInvitation(ctx, inv.ID, actor.UserID, u.clock.Now())
	})
}
