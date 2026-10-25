package bootstrap

import (
	"context"
	"slices"
	"uuid"

	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// fakeIdentityProfiles answers the profiles it holds of the ids asked for,
// in the order it holds them, and records the ids and context.
type fakeIdentityProfiles struct {
	profiles []identityapp.PublicProfile
	err      error
	asked    [][]uuid.UUID
	ctx      context.Context
}

func (f *fakeIdentityProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]identityapp.PublicProfile, error) {
	f.ctx = ctx
	f.asked = append(f.asked, ids)
	var out []identityapp.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, f.err
}
