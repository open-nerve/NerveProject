package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActivateDeps are Activate's collaborators.
type ActivateDeps struct {
	Accounts  AccountLocker
	Users     UserActivator
	APITokens UsableAPITokenCounter
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// Activate activates an account for the server's administrator:
// `nerve users activate` (M2 decision 3, design 3.17).
type Activate struct {
	d ActivateDeps
}

// NewActivate returns the use case.
func NewActivate(d ActivateDeps) *Activate {
	return &Activate{d: d}
}

// ActivateResult is the account's address and how many of its personal
// access tokens authenticate again.
type ActivateResult struct {
	Email     string // normalized
	APITokens int
}

// Execute activates the account with email in one transaction: it locks the
// account row by the address (identity.account_not_found), sets it active
// and counts the tokens that are unrevoked and unexpired. Deactivation kept
// them, so they authenticate again; when the account may be compromised,
// reset-password revokes them (M2 design 3.17).
func (u *Activate) Execute(ctx context.Context, email string) (ActivateResult, error) {
	email = domain.NormalizeEmail(email)
	now := u.d.Clock.Now()
	result := ActivateResult{Email: email}
	var id uuid.UUID
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		if err := u.d.Users.ActivateUser(ctx, id, now); err != nil {
			return err
		}
		result.APITokens, err = u.d.APITokens.CountUsableAPITokens(ctx, id, now)
		return err
	})
	if err != nil {
		return ActivateResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "account activated", slog.String("user_id", id.String()),
		slog.Int("usable_api_tokens", result.APITokens), slog.String("by", byCLI))
	return result, nil
}
