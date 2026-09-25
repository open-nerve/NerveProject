package app

import (
	"context"
	"crypto/rand"
	"log/slog"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Tokens are what registration (and from M2/P2 login and refresh) returns.
type Tokens struct {
	AccessToken      string
	AccessExpiresIn  time.Duration
	RefreshToken     string
	RefreshExpiresAt time.Time // the session's absolute end (M2 design 3.5)
}

// RegisterDeps are Register's collaborators and settings.
type RegisterDeps struct {
	Policy     SignupPolicy
	Rules      *domain.PasswordRules
	Hasher     PasswordHasher
	Tx         shared.TxManager
	Users      UserCreator
	Profiles   ProfileCreator
	Sessions   SessionCreator
	Tokens     AccessTokens
	MAC        RefreshTokenMAC
	Clock      Clock
	Logger     *slog.Logger
	AccessTTL  time.Duration // auth.access_token_ttl
	SessionTTL time.Duration // auth.session_ttl
}

// Register creates an account and signs it in: POST /api/v0/auth/register.
type Register struct {
	d        RegisterDeps
	accounts accounts
}

// NewRegister returns the use case.
func NewRegister(d RegisterDeps) *Register {
	return &Register{d: d, accounts: accounts{users: d.Users, profiles: d.Profiles}}
}

// RegisterInput is a registration and where it comes from.
type RegisterInput struct {
	Email     string
	Password  string
	UserAgent string
	IP        netip.Addr
}

// Execute registers in.Email:
//
//  1. closed sign-up answers 403 before any other check (M2 design 3.9);
//  2. the address and the password are validated, all fields at once (422);
//  3. the password is hashed outside the transaction (M2 design 3.5);
//  4. one transaction inserts the account, its profile and its first
//     session; an address in use is 409 identity.email_taken.
//
// The tokens are made before the transaction, so nothing can fail after it
// commits.
func (r *Register) Execute(ctx context.Context, in RegisterInput) (Tokens, error) {
	allowed, err := r.d.Policy.AllowSignup(ctx)
	if err != nil {
		return Tokens{}, err
	}
	if !allowed {
		return Tokens{}, domain.ErrSignupDisabled
	}
	email, err := domain.NewAccount(r.d.Rules, in.Email, in.Password)
	if err != nil {
		return Tokens{}, err
	}
	hash, err := r.d.Hasher.Hash(ctx, in.Password)
	if err != nil {
		return Tokens{}, err
	}

	now := r.d.Clock.Now()
	user := NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: now}
	refresh := domain.RefreshToken{SessionID: uuid.NewV7(), Generation: 0}
	_, _ = rand.Read(refresh.Secret[:]) // never fails since Go 1.24
	refresh.Tag = r.d.MAC.Tag(refresh.MACMessage())
	session := NewSession{
		ID:        refresh.SessionID,
		UserID:    user.ID,
		TokenHash: refresh.SecretHash(),
		UserAgent: domain.SanitizeUserAgent(in.UserAgent),
		IP:        in.IP,
		ExpiresAt: now.Add(r.d.SessionTTL),
		Now:       now,
	}
	access, err := r.d.Tokens.Issue(AccessClaims{UserID: user.ID, SessionID: session.ID, ExpiresAt: now.Add(r.d.AccessTTL)})
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
	return Tokens{
		AccessToken:      access,
		AccessExpiresIn:  r.d.AccessTTL,
		RefreshToken:     refresh.String(),
		RefreshExpiresAt: session.ExpiresAt,
	}, nil
}
