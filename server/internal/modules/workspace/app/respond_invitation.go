package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// responder is what accepting and declining an invitation share (M3 design
// 3.8): the caller answers for himself, account level, without the
// Authorizer (6.4). The link's token is checked before anything is read, so
// the answer to a wrong one cannot depend on the invitation's existence.
// Then, in the transaction, in the order of 3.6: the caller's account row
// FOR SHARE, the first lock (conventions 1 and 6), its state and address
// read under that lock; the invitation read for its workspace; the
// workspace, by the lock the response passes; the invitation FOR UPDATE,
// read again. Only then are the addresses compared, the one read under the
// account's lock with the invitation's, then whether the invitation was
// answered: an address that is not the invitation's writes nothing.
type responder struct {
	accounts    Accounts
	invitations InvitationLocker
	tokens      invitationTokens
}

// checkToken is workspace.invitation_not_found unless token is the
// invitation id's.
func (r responder) checkToken(id uuid.UUID, token string) error {
	if !r.tokens.valid(id, token) {
		return domain.ErrInvitationNotFound
	}
	return nil
}

// lock takes the locks in the transaction ctx carries and returns the
// caller's account and the invitation as read under them: 401 unauthorized
// for an account deactivated or gone, workspace.invitation_not_found for an
// invitation not there or deleted meanwhile, or whose workspace is,
// workspace.invitation_email_mismatch for another address,
// workspace.invitation_responded for a declined invitation.
func (r responder) lock(ctx context.Context, actor shared.Actor, id uuid.UUID,
	lockWorkspace func(ctx context.Context, id uuid.UUID) error) (AccountState, domain.Invitation, error) {
	account, found, err := r.accounts.ShareAccount(ctx, actor.UserID)
	switch {
	case err != nil:
		return AccountState{}, domain.Invitation{}, err
	case !found || !account.Active:
		return AccountState{}, domain.Invitation{}, shared.Unauthenticated()
	}
	inv, err := readInvitation(ctx, r.invitations, id)
	if err != nil {
		return AccountState{}, domain.Invitation{}, err
	}
	if inv, err = lockInvitation(ctx, r.invitations, lockWorkspace, inv); err != nil {
		return AccountState{}, domain.Invitation{}, err
	}
	switch {
	case account.Email != inv.Email:
		return AccountState{}, domain.Invitation{}, domain.ErrInvitationEmailMismatch
	case inv.Responded():
		return AccountState{}, domain.Invitation{}, domain.ErrInvitationResponded
	}
	return account, inv, nil
}
