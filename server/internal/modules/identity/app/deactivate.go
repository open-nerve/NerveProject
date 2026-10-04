package app

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeactivateDeps are Deactivate's collaborators. Execute locks with Lock,
// ExecuteByEmail with Accounts: a composition sets the one its entry uses.
// Memberships is the workspace module's MembershipDeactivator, which both
// call.
type DeactivateDeps struct {
	Lock        CredentialLock
	Accounts    AccountLocker
	Users       UserDeactivator
	Profiles    OnboardingResetter
	Sessions    SessionRevoker
	Memberships MembershipDeactivator
	Tx          shared.TxManager
	Clock       Clock
	Logger      *slog.Logger
}

// Deactivate deactivates an account: the caller's own,
// POST /api/v0/me/deactivate, or one the server's administrator names,
// `nerve users deactivate` (M2 decision 3). The caller may use any
// credential, a token too, and no password is asked for, as in Plane
// (M2 design 8.6).
type Deactivate struct {
	d DeactivateDeps
}

// NewDeactivate returns the use case.
func NewDeactivate(d DeactivateDeps) *Deactivate {
	return &Deactivate{d: d}
}

// Execute deactivates the caller's account: it takes the credential lock,
// then writes as deactivate does, with the address the lock read.
func (u *Deactivate) Execute(ctx context.Context) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	now := u.d.Clock.Now()
	var revoked int
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		account, err := u.d.Lock.Lock(ctx, actor, now)
		if err != nil {
			return err
		}
		revoked, err = u.deactivate(ctx, actor.UserID, account.Email, now)
		return err
	})
	if err != nil {
		return err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", actor.UserID.String()),
		slog.Int("revoked_sessions", revoked), slog.String("by", "self"))
	return nil
}

// DeactivateResult is the account's address and how many sessions the
// deactivation revoked.
type DeactivateResult struct {
	Email    string // normalized
	Sessions int
}

// ExecuteByEmail deactivates the account with email for the server's
// administrator: it locks the account row by the address
// (identity.account_not_found), then writes as deactivate does, with that
// address, the row's under the lock.
func (u *Deactivate) ExecuteByEmail(ctx context.Context, email string) (DeactivateResult, error) {
	email = shared.NormalizeEmail(email)
	now := u.d.Clock.Now()
	result := DeactivateResult{Email: email}
	var id uuid.UUID
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if id, err = lockAccount(ctx, u.d.Accounts, email); err != nil {
			return err
		}
		result.Sessions, err = u.deactivate(ctx, id, email, now)
		return err
	})
	if err != nil {
		return DeactivateResult{}, err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", id.String()),
		slog.Int("revoked_sessions", result.Sessions), slog.String("by", byCLI))
	return result, nil
}

// deactivate writes in the global lock order (M2 design 3.5, 6.4; M3 design
// 3.6): the account inactive, its onboarding started over, every session
// revoked with reason deactivated, then, last, its memberships ended and
// the invitations to email, its address under the lock, deleted (M3 design
// 3.9); it returns how many sessions. The password and the personal access
// tokens stay; authentication refuses the tokens while the account is
// inactive. Call it under the account row lock.
func (u *Deactivate) deactivate(ctx context.Context, id uuid.UUID, email string, now time.Time) (int, error) {
	if err := u.d.Users.DeactivateUser(ctx, id, now); err != nil {
		return 0, err
	}
	if err := u.d.Profiles.ResetOnboarding(ctx, id, now); err != nil {
		return 0, err
	}
	revoked, err := u.d.Sessions.RevokeSessions(ctx, id, uuid.Nil(), domain.RevokeDeactivated, now)
	if err != nil {
		return 0, err
	}
	return revoked, u.d.Memberships.DeactivateMemberships(ctx, id, email)
}
