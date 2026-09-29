package app

import (
	"context"
	"errors"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ResetPasswordDeps are ResetPassword's collaborators.
type ResetPasswordDeps struct {
	Accounts  AccountLocker
	Passwords PasswordHashWriter
	Sessions  SessionRevoker
	APITokens AllAPITokensRevoker
	Hasher    PasswordHasher
	Rules     *domain.PasswordRules
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// ResetPassword sets an account's password for the server's administrator:
// `nerve users reset-password`, the way back into an account whose
// credentials leaked (M2 design 3.5, 3.17, 8.5).
type ResetPassword struct {
	d ResetPasswordDeps
}

// NewResetPassword returns the use case.
func NewResetPassword(d ResetPasswordDeps) *ResetPassword {
	return &ResetPassword{d: d}
}

// ResetPasswordResult is the account's address and what the reset revoked.
type ResetPasswordResult struct {
	Email     string // normalized
	Sessions  int
	APITokens int
}

// Execute resets the password of the account with email:
//
//  1. the new password is checked by the password rules, with the address
//     for the stem rule: 422 validation_failed on password;
//  2. it is hashed outside the transaction (M2 design 3.5);
//  3. one transaction locks the account row by its address, writes the
//     hash, then revokes every session (password_reset) and every personal
//     access token, in the global lock order. No such account is
//     identity.account_not_found.
//
// A login or a token creation that verified the old password holds or waits
// for the same lock, and finds its credential revoked (interleavings 1-3).
func (u *ResetPassword) Execute(ctx context.Context, email, password string) (ResetPasswordResult, error) {
	email = shared.NormalizeEmail(email)
	if f := u.d.Rules.Check("password", password, email); f != nil {
		return ResetPasswordResult{}, shared.Invalid(*f)
	}
	hash, err := u.d.Hasher.Hash(ctx, password)
	if err != nil {
		return ResetPasswordResult{}, err
	}
	now := u.d.Clock.Now()
	result := ResetPasswordResult{Email: email}
	var id uuid.UUID
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		if err := u.d.Passwords.UpdatePasswordHash(ctx, id, hash, now); err != nil {
			return err
		}
		if result.Sessions, err = u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokePasswordReset, now); err != nil {
			return err
		}
		result.APITokens, err = u.d.APITokens.RevokeAllAPITokens(ctx, id, now)
		return err
	})
	if err != nil {
		return ResetPasswordResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "password reset", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.Int("revoked_api_tokens", result.APITokens), slog.String("by", byCLI))
	return result, nil
}

// lockAccount takes the account row lock by address: identity.account_not_found
// when no account has it. Call it first inside the transaction.
func lockAccount(ctx context.Context, accounts AccountLocker, email string) (uuid.UUID, error) {
	id, err := accounts.LockAccount(ctx, email)
	if errors.Is(err, ErrNotFound) {
		return uuid.Nil(), domain.ErrAccountNotFound
	}
	return id, err
}
