package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeclineWorkspaceInvitation answers an invitation to the caller's address
// with no: POST /api/v0/workspace-invitations/{invitation_id}/decline.
type DeclineWorkspaceInvitation struct {
	invitations InvitationDecliner
	tx          shared.TxManager
	clock       Clock
	responder   responder
}

// NewDeclineWorkspaceInvitation returns the use case.
func NewDeclineWorkspaceInvitation(accounts Accounts, invitations InvitationDecliner, tx shared.TxManager, clock Clock,
	mac InvitationMAC) *DeclineWorkspaceInvitation {
	return &DeclineWorkspaceInvitation{invitations: invitations, tx: tx, clock: clock,
		responder: responder{accounts: accounts, invitations: invitations, tokens: invitationTokens{mac: mac}}}
}

// Execute takes responder's locks, the workspace FOR SHARE: declining adds
// no membership, and the account's lock comes first all the same, so the
// two answers check the address alike and neither locks the account after
// a workspace (M3 design 3.8, 3.6 convention 1). Then it reads the clock
// and records the answer. The invitation stays, declined, and holds its
// address until an admin deletes it.
func (u *DeclineWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	if err := u.responder.checkToken(id, token); err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		account, inv, err := u.responder.lock(ctx, actor, id, u.invitations.ShareWorkspace)
		if err != nil {
			return err
		}
		return u.invitations.DeclineInvitation(ctx, inv.ID, account.ID, u.clock.Now())
	})
}
