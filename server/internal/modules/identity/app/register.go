package app

import (
	"context"
	"log/slog"
	"net/netip"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RegisterDeps are Register's collaborators and settings.
type RegisterDeps struct {
	Policy   SignupPolicy
	Rules    *domain.PasswordRules
	Hasher   PasswordHasher
	Tx       shared.TxManager
	Users    UserCreator
	Profiles ProfileCreator
	Sessions SessionCreator
	Issuance Issuance
	Clock    Clock
	Logger   *slog.Logger
}

// Register creates an account and signs it in: POST /api/v0/auth/register.
type Register struct {
	d        RegisterDeps
	accounts accounts
}

// NewRegister returns the use case.
func NewRegister(d RegisterDeps) *Register {
	return &Register{d: d, accounts: accounts{rules: d.Rules, hasher: d.Hasher, clock: d.Clock, users: d.Users, profiles: d.Profiles}}
}

// RegisterInput is a registration and where it comes from.
type RegisterInput struct {
	Email      string
	Password   string
	Invitation *SignupInvitation // the invitation the request names; nil when none
	UserAgent  string
	IP         netip.Addr
}

// Execute registers in.Email:
//
//  1. the policy decides on the address, normalized, and the invitation,
//     before any other check: a refusal answers 403 for every address and
//     every invitation alike (M2 design 3.9, M3 design 3.8);
//  2. the address and the password are validated, all fields at once (422);
//  3. the password is hashed outside the transaction (M2 design 3.5);
//  4. one transaction inserts the account, its profile and its first
//     session; an address in use is 409 identity.email_taken.
//
// The tokens are made before the transaction, so nothing can fail after it
// commits.
func (r *Register) Execute(ctx context.Context, in RegisterInput) (Tokens, error) {
	allowed, err := r.d.Policy.AllowSignup(ctx, shared.NormalizeEmail(in.Email), in.Invitation)
	if err != nil {
		return Tokens{}, err
	}
	if !allowed {
		return Tokens{}, domain.ErrSignupDisabled
	}
	user, err := r.accounts.prepare(ctx, in.Email, in.Password)
	if err != nil {
		return Tokens{}, err
	}
	session, tokens, err := r.d.Issuance.newSession(user.ID, in.UserAgent, in.IP, user.Now)
	if err != nil {
		return Tokens{}, err
	}

	err = r.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := r.accounts.create(ctx, user); err != nil {
			return err
		}
		return r.d.Sessions.CreateSession(ctx, session)
	})
	if err != nil {
		return Tokens{}, err
	}
	r.d.Logger.InfoContext(ctx, "account registered", slog.String("user_id", user.ID.String()))
	return tokens, nil
}
