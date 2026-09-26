package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetProfile reads the caller's preferences: GET /api/v0/me/profile.
type GetProfile struct {
	profiles ProfileReader
}

// NewGetProfile returns the use case.
func NewGetProfile(profiles ProfileReader) *GetProfile {
	return &GetProfile{profiles: profiles}
}

// Execute returns the preferences of the request's actor.
func (g *GetProfile) Execute(ctx context.Context) (domain.Profile, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Profile{}, err
	}
	p, err := g.profiles.GetProfile(ctx, actor.UserID)
	if errors.Is(err, ErrNotFound) {
		// Every account has its profile from registration on (M2 design
		// 4.3), and it goes only with the account: the account is gone.
		return domain.Profile{}, shared.Unauthenticated()
	}
	return p, err
}
