package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PreferencesChange is a change of an account's display settings on a
// page: the id a row inserted for it takes; the page's workspace, project,
// and cycle or module, Page; the account; the change; and its moment.
type PreferencesChange[F domain.PageFilters] struct {
	ID, WorkspaceID, ProjectID, Page, UserID uuid.UUID
	Patch                                    domain.PreferencesPatch[F]
	Now                                      time.Time
}

// PreferencesWriter changes an account's display settings on a cycle's or a
// module's page, in the transaction ctx carries, which holds the page's
// project FOR SHARE: his undeleted row has the fields the change gives
// replaced, or, while he has none, one is inserted with the defaults and the
// change applied; made or last changed by him at the change's moment. The
// answer is the settings as stored.
type PreferencesWriter interface {
	UpsertCyclePreferences(ctx context.Context, c PreferencesChange[domain.CycleFilters]) (domain.CyclePreferences, error)
	UpsertModulePreferences(ctx context.Context, c PreferencesChange[domain.ModuleFilters]) (domain.ModulePreferences, error)
}

// UpdatePreferencesDeps are what changing one's display settings on a page
// needs.
type UpdatePreferencesDeps struct {
	Locks       Locks
	Settings    IssueListSettings
	Clock       Clock
	Preferences PreferencesWriter
}

// UpdateCyclePreferences changes the caller's display settings on a cycle's
// page: PATCH /api/v0/me/cycles/{cycle_id}/preferences.
type UpdateCyclePreferences struct{ d UpdatePreferencesDeps }

// NewUpdateCyclePreferences returns the use case.
func NewUpdateCyclePreferences(d UpdatePreferencesDeps) *UpdateCyclePreferences {
	return &UpdateCyclePreferences{d: d}
}

// Execute changes the caller's settings on the cycle's page
// (changePreferences).
func (u *UpdateCyclePreferences) Execute(ctx context.Context, id uuid.UUID, p domain.CyclePreferencesPatch) (domain.CyclePreferences, error) {
	return changePreferences(ctx, u.d, domain.ActionCyclePreferences, named{cycle: &id}, id, p, u.d.Preferences.UpsertCyclePreferences)
}

// UpdateModulePreferences changes the caller's display settings on a
// module's page: PATCH /api/v0/me/modules/{module_id}/preferences.
type UpdateModulePreferences struct{ d UpdatePreferencesDeps }

// NewUpdateModulePreferences returns the use case.
func NewUpdateModulePreferences(d UpdatePreferencesDeps) *UpdateModulePreferences {
	return &UpdateModulePreferences{d: d}
}

// Execute changes the caller's settings on the module's page
// (changePreferences).
func (u *UpdateModulePreferences) Execute(ctx context.Context, id uuid.UUID, p domain.ModulePreferencesPatch) (domain.ModulePreferences, error) {
	return changePreferences(ctx, u.d, domain.ActionModulePreferences, named{module: &id}, id, p, u.d.Preferences.UpsertModulePreferences)
}

// changePreferences checks the words p gives, a refusal answered before
// anything is read (checkListSettings); then, in one transaction, takes the
// lock of the page n names and the decision on action (Locks.share), which
// refuses no OwnOnly grant, archived project or toggle turned off (M6 P3b
// spec R6); reads the clock under the lock; and changes the caller's own
// settings on page, or inserts them (upsert). The answer is the settings as
// stored.
func changePreferences[F domain.PageFilters](ctx context.Context, d UpdatePreferencesDeps, action shared.Action, n named, page uuid.UUID,
	p domain.PreferencesPatch[F], upsert func(context.Context, PreferencesChange[F]) (domain.Preferences[F], error)) (domain.Preferences[F], error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Preferences[F]{}, err
	}
	if err := checkListSettings(p, d.Settings); err != nil {
		return domain.Preferences[F]{}, err
	}
	var stored domain.Preferences[F]
	err = d.Locks.share(ctx, actor, action, n, func(ctx context.Context, h held) error {
		now := d.Clock.Now()
		var err error
		stored, err = upsert(ctx, PreferencesChange[F]{ID: uuid.NewV7(), WorkspaceID: h.project.WorkspaceID, ProjectID: h.projectID, Page: page,
			UserID: actor.UserID, Patch: p, Now: now})
		return err
	})
	if err != nil {
		return domain.Preferences[F]{}, err
	}
	return stored, nil
}
