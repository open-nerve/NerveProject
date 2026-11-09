package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PreferencesReader reads an account's undeleted display settings on a
// cycle's or a module's page, in the transaction ctx carries; found is
// false while he has no row of them.
type PreferencesReader interface {
	CyclePreferences(ctx context.Context, cycle, user uuid.UUID) (p domain.CyclePreferences, found bool, err error)
	ModulePreferences(ctx context.Context, module, user uuid.UUID) (p domain.ModulePreferences, found bool, err error)
}

// GetPreferencesDeps are what reading one's display settings on a page
// needs.
type GetPreferencesDeps struct {
	Reads       Reads
	Preferences PreferencesReader
}

// GetCyclePreferences reads the caller's display settings on a cycle's
// page: GET /api/v0/me/cycles/{cycle_id}/preferences.
type GetCyclePreferences struct{ d GetPreferencesDeps }

// NewGetCyclePreferences returns the use case.
func NewGetCyclePreferences(d GetPreferencesDeps) *GetCyclePreferences {
	return &GetCyclePreferences{d: d}
}

// Execute answers the caller's settings on the cycle's page (readPreferences).
func (u *GetCyclePreferences) Execute(ctx context.Context, id uuid.UUID) (domain.CyclePreferences, error) {
	return readPreferences(ctx, u.d.Reads, domain.ActionCyclePreferences, named{cycle: &id}, id, u.d.Preferences.CyclePreferences)
}

// GetModulePreferences reads the caller's display settings on a module's
// page: GET /api/v0/me/modules/{module_id}/preferences.
type GetModulePreferences struct{ d GetPreferencesDeps }

// NewGetModulePreferences returns the use case.
func NewGetModulePreferences(d GetPreferencesDeps) *GetModulePreferences {
	return &GetModulePreferences{d: d}
}

// Execute answers the caller's settings on the module's page
// (readPreferences).
func (u *GetModulePreferences) Execute(ctx context.Context, id uuid.UUID) (domain.ModulePreferences, error) {
	return readPreferences(ctx, u.d.Reads, domain.ActionModulePreferences, named{module: &id}, id, u.d.Preferences.ModulePreferences)
}

// readPreferences decides action on the page n names in a read's snapshot
// (Reads), whatever the caller's grant, the project's archive or its toggles
// (M6 P3b spec R6), and answers the caller's settings on page, read there,
// or domain.DefaultPreferences while he has no row of them. A read writes
// nothing: the first change makes the row.
func readPreferences[F domain.PageFilters](ctx context.Context, reads Reads, action shared.Action, n named, page uuid.UUID,
	read func(ctx context.Context, page, user uuid.UUID) (domain.Preferences[F], bool, error)) (domain.Preferences[F], error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences[F]{}, err
	}
	out := domain.DefaultPreferences[F]()
	err = reads.run(ctx, actor, action, n, func(ctx context.Context, _ held) error {
		p, found, err := read(ctx, page, actor.UserID)
		switch {
		case err != nil:
			return err
		case found:
			out = p
		}
		return nil
	})
	if err != nil {
		return domain.Preferences[F]{}, err
	}
	return out, nil
}
