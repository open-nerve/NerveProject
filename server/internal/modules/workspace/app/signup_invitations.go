package app

import (
	"context"
	"errors"
	"uuid"
)

// SignupInvitations checks the invitation a registration names while
// sign-up is closed (M3 design 3.8, decision 1): bootstrap's signup policy
// asks it for identity's registration. It reads; it accepts nothing.
type SignupInvitations struct {
	invitations InvitationFinder
	tokens      invitationTokens
}

// NewSignupInvitations returns the check.
func NewSignupInvitations(invitations InvitationFinder, mac InvitationMAC) *SignupInvitations {
	return &SignupInvitations{invitations: invitations, tokens: invitationTokens{mac: mac}}
}

// Allows reports whether token is the link of the invitation id, pending,
// to email, normalized as registration normalizes it. The token is checked
// first and reads nothing. Every other case is the same false, which the
// registration answers with the same identity.signup_disabled (M3 design
// 8.2): a wrong token, an invitation not there, deleted, accepted or
// declined, another address. A failed read is an error, never a refusal.
func (s *SignupInvitations) Allows(ctx context.Context, email string, id uuid.UUID, token string) (bool, error) {
	if !s.tokens.valid(id, token) {
		return false, nil
	}
	inv, err := s.invitations.InvitationByID(ctx, id)
	switch {
	case errors.Is(err, ErrNotFound):
		return false, nil
	case err != nil:
		return false, err
	}
	return !inv.Responded() && inv.Email == email, nil
}
