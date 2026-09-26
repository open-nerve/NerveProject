package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeactivateDeps are Deactivate's collaborators.
type DeactivateDeps struct {
	Lock     CredentialLock
	Users    UserDeactivator
	Profiles OnboardingResetter
	Sessions SessionRevoker
	Tx       shared.TxManager
	Clock    Clock
	Logger   *slog.Logger
}

// Deactivate deactivates the caller's account: POST /api/v0/me/deactivate.
// Any credential may, a token too, and no password is asked for, as in
// Plane (M2 decision 3, design 8.6).
type Deactivate struct {
	d DeactivateDeps
}

// NewDeactivate returns the use case.
func NewDeactivate(d DeactivateDeps) *Deactivate {
	return &Deactivate{d: d}
}

// Execute takes the credential lock, then writes in the global lock order
// (M2 design 3.5, 6.4): the account inactive, its onboarding started over,
// every session revoked with reason deactivated. The password and the
// personal access tokens stay; authentication refuses the tokens while the
// account is inactive.
func (u *Deactivate) Execute(ctx context.Context) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	now := u.d.Clock.Now()
	var revoked int
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := u.d.Lock.Lock(ctx, actor, now); err != nil {
			return err
		}
		if err := u.d.Users.DeactivateUser(ctx, actor.UserID, now); err != nil {
			return err
		}
		if err := u.d.Profiles.ResetOnboarding(ctx, actor.UserID, now); err != nil {
			return err
		}
		var err error
		revoked, err = u.d.Sessions.RevokeSessions(ctx, actor.UserID, uuid.Nil(), domain.RevokeDeactivated, now)
		return err
	})
	if err != nil {
		return err
	}
	u.d.Logger.InfoContext(ctx, "account deactivated", slog.String("user_id", actor.UserID.String()),
		slog.Int("revoked_sessions", revoked), slog.String("by", "self"))
	return nil
}
