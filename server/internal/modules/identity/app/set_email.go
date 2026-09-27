package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SetEmailDeps are SetEmail's collaborators.
type SetEmailDeps struct {
	Accounts AccountLocker
	Users    EmailChanger
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
}

// SetEmail changes an account's address for the server's administrator:
// `nerve users set-email`, the only way to change it (M2 decision 1,
// design 3.17).
type SetEmail struct {
	d SetEmailDeps
}

// NewSetEmail returns the use case.
func NewSetEmail(d SetEmailDeps) *SetEmail {
	return &SetEmail{d: d}
}

// SetEmailResult is the old and the new address, normalized, and how many
// sessions the change revoked.
type SetEmailResult struct {
	From, To string
	Sessions int
}

// Execute changes the address email to newEmail:
//
//  1. both are normalized; the new one is checked by the rules of
//     registration (422 validation_failed on new_email), and must differ
//     from the old one (identity.email_unchanged);
//  2. one transaction locks the account row by the old address
//     (identity.account_not_found), writes the new one
//     (identity.email_taken when another account has it) and revokes every
//     session (email_changed).
//
// The personal access tokens stay: changing the address is no recovery
// from a leak; reset-password is (M2 design 3.17).
func (u *SetEmail) Execute(ctx context.Context, email, newEmail string) (SetEmailResult, error) {
	from := domain.NormalizeEmail(email)
	to, err := domain.NewEmail("new_email", newEmail)
	if err != nil {
		return SetEmailResult{}, err
	}
	if to == from {
		return SetEmailResult{}, domain.ErrEmailUnchanged
	}
	now := u.d.Clock.Now()
	result := SetEmailResult{From: from, To: to}
	var id uuid.UUID
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, from); err != nil {
			return err
		}
		if err := u.d.Users.ChangeEmail(ctx, id, to, now); err != nil {
			return err
		}
		result.Sessions, err = u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeEmailChanged, now)
		return err
	})
	if err != nil {
		return SetEmailResult{}, err
	}
	// Neither address is logged (M2 design 8.4).
	u.d.Logger.InfoContext(ctx, "e-mail address changed", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.String("by", byCLI))
	return result, nil
}
