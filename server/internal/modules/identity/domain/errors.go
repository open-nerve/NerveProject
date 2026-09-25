package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The identity module's errors (M2 design 5.4). api/modules/identity.yaml
// declares their codes in x-problem-codes.
var (
	// ErrSignupDisabled answers a well-formed registration while sign-up is
	// off, before the address or the password is looked at (M2 design 3.9);
	// the platform's structural 400 or 413 can come first.
	ErrSignupDisabled = shared.NewError(shared.KindForbidden, "identity.signup_disabled", "Sign-up is disabled on this instance.")
	// ErrEmailTaken answers a registration with an address in use.
	ErrEmailTaken = shared.NewError(shared.KindConflict, "identity.email_taken", "An account with this e-mail address already exists.")
)
