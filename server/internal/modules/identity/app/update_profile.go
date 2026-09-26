package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProfile changes the caller's preferences: PATCH /api/v0/me/profile.
// It is one statement, which merges the onboarding steps (M2 design 3.14,
// 6.4).
type UpdateProfile struct {
	profiles ProfileUpdater
	clock    Clock
}

// NewUpdateProfile returns the use case.
func NewUpdateProfile(profiles ProfileUpdater, clock Clock) *UpdateProfile {
	return &UpdateProfile{profiles: profiles, clock: clock}
}

// Execute checks p and applies it to the caller's preferences at the
// clock's now; the fields and steps p leaves nil stay as they are.
func (u *UpdateProfile) Execute(ctx context.Context, p domain.ProfilePatch) (domain.Profile, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	if err := domain.CheckProfilePatch(p); err != nil {
		return domain.Profile{}, err
	}
	profile, err := u.profiles.UpdateProfile(ctx, actor.UserID, p, u.clock.Now())
	if errors.Is(err, ErrNotFound) {
		// The profile goes only with the account (GetProfile).
		return domain.Profile{}, shared.Unauthenticated()
	}
	return profile, err
}
