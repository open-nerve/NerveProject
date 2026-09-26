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
	// ErrInvalidCredentials answers a login with an unknown address or a
	// wrong password alike (M2 design 3.9).
	ErrInvalidCredentials = shared.NewError(shared.KindUnauthenticated, "identity.invalid_credentials", "The e-mail address or the password is incorrect.")
	// ErrAccountDeactivated answers a login with the right password for a
	// deactivated account; only then is the state revealed (M2 design 3.9).
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "identity.account_deactivated", "This account is deactivated.")
	// ErrRefreshTokenInvalid answers every refresh that does not rotate:
	// unknown, expired, revoked, reused or forged (M2 design 3.5).
	ErrRefreshTokenInvalid = shared.NewError(shared.KindUnauthenticated, "identity.refresh_token_invalid", "The refresh token is not valid; sign in again.")
	// ErrCurrentPasswordIncorrect answers a change of password whose current
	// password is wrong, or was changed concurrently since it was verified
	// (M2 design 3.5).
	ErrCurrentPasswordIncorrect = shared.NewError(shared.KindInvalid, "identity.current_password_incorrect", "The current password is incorrect.")
	// ErrAPITokenNotFound answers a revocation of a token that does not
	// exist, is revoked already or belongs to another account: what the
	// caller cannot see is not found (v0 design 3.5).
	ErrAPITokenNotFound = shared.NewError(shared.KindNotFound, "identity.api_token_not_found", "The API token does not exist.")
)
