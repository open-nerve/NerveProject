package bootstrap

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// signupPolicy is identity's SignupPolicy (M3 design 3.8, 6.5):
// auth.signup_enabled; while it is off, a registration goes on only with an
// invitation workspace allows for its address. Neither module knows the
// other: the switch and the check meet here.
type signupPolicy struct {
	enabled     bool
	invitations workspace.SignupInvitations
}

// AllowSignup allows every registration while sign-up is on, without
// asking about the invitation; while it is off, none without one, and one
// with one as workspace answers.
func (p signupPolicy) AllowSignup(ctx context.Context, email string, invitation *identity.SignupInvitation) (bool, error) {
	switch {
	case p.enabled:
		return true, nil
	case invitation == nil:
		return false, nil
	}
	return p.invitations.Allows(ctx, email, invitation.ID, invitation.Token)
}
