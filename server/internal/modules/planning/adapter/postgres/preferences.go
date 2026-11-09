package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// DeleteCyclePreferenceRows soft-deletes the deleted cycle's page's undeleted
// display settings, each account's row, locked in id order, deleted and last
// changed by by at at, in one statement (preferences.sql).
func (s *Store) DeleteCyclePreferenceRows(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteCyclePreferenceRows(ctx, gen.DeleteCyclePreferenceRowsParams{CycleID: cycle, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the display settings of cycle %s: %w", cycle, err)
	}
	return nil
}

// DeleteModulePreferenceRows is DeleteCyclePreferenceRows of the deleted
// module's page.
func (s *Store) DeleteModulePreferenceRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteModulePreferenceRows(ctx, gen.DeleteModulePreferenceRowsParams{ModuleID: module, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the display settings of module %s: %w", module, err)
	}
	return nil
}

// CyclePreferences reads the account's undeleted display settings on the
// cycle's page; found is false while he has none (app.PreferencesReader).
func (s *Store) CyclePreferences(ctx context.Context, cycle, user uuid.UUID) (domain.CyclePreferences, bool, error) {
	r, err := s.queries(ctx).CyclePreferences(ctx, gen.CyclePreferencesParams{CycleID: cycle, UserID: user})
	return foundPreferences[domain.CycleFilters](r.Filters, r.DisplayFilters, r.DisplayProperties, err, "cycle", cycle)
}

// ModulePreferences reads the account's undeleted display settings on the
// module's page; found is false while he has none (app.PreferencesReader).
func (s *Store) ModulePreferences(ctx context.Context, module, user uuid.UUID) (domain.ModulePreferences, bool, error) {
	r, err := s.queries(ctx).ModulePreferences(ctx, gen.ModulePreferencesParams{ModuleID: module, UserID: user})
	return foundPreferences[domain.ModuleFilters](r.Filters, r.DisplayFilters, r.DisplayProperties, err, "module", module)
}

// UpsertCyclePreferences applies c.Patch to the account's undeleted row of
// the cycle's page, or inserts one with domain.DefaultPreferences and the
// patch applied, made or last changed by him at c.Now, and answers the
// settings as stored (app.PreferencesWriter).
func (s *Store) UpsertCyclePreferences(ctx context.Context, c app.PreferencesChange[domain.CycleFilters]) (domain.CyclePreferences, error) {
	v, err := preferencesValues(c)
	if err != nil {
		return domain.CyclePreferences{}, err
	}
	r, err := s.queries(ctx).UpsertCyclePreferences(ctx, gen.UpsertCyclePreferencesParams{
		ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, CycleID: c.Page, UserID: c.UserID, Now: c.Now,
		Filters: v.filters, DisplayFilters: v.displayFilters, DisplayProperties: v.displayProperties,
		SetFilters: c.Patch.Filters != nil, SetDisplayFilters: c.Patch.DisplayFilters != nil, SetDisplayProperties: c.Patch.DisplayProperties != nil,
	})
	if err != nil {
		return domain.CyclePreferences{}, fmt.Errorf("write the display settings on cycle %s: %w", c.Page, err)
	}
	return preferencesOf[domain.CycleFilters](r.Filters, r.DisplayFilters, r.DisplayProperties)
}

// UpsertModulePreferences is UpsertCyclePreferences of the module's page.
func (s *Store) UpsertModulePreferences(ctx context.Context, c app.PreferencesChange[domain.ModuleFilters]) (domain.ModulePreferences, error) {
	v, err := preferencesValues(c)
	if err != nil {
		return domain.ModulePreferences{}, err
	}
	r, err := s.queries(ctx).UpsertModulePreferences(ctx, gen.UpsertModulePreferencesParams{
		ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, ModuleID: c.Page, UserID: c.UserID, Now: c.Now,
		Filters: v.filters, DisplayFilters: v.displayFilters, DisplayProperties: v.displayProperties,
		SetFilters: c.Patch.Filters != nil, SetDisplayFilters: c.Patch.DisplayFilters != nil, SetDisplayProperties: c.Patch.DisplayProperties != nil,
	})
	if err != nil {
		return domain.ModulePreferences{}, fmt.Errorf("write the display settings on module %s: %w", c.Page, err)
	}
	return preferencesOf[domain.ModuleFilters](r.Filters, r.DisplayFilters, r.DisplayProperties)
}

// preferenceColumns are a page's settings as their three jsonb columns.
type preferenceColumns struct{ filters, displayFilters, displayProperties []byte }

// preferencesValues are the columns of the row c inserts: the defaults with
// its change applied. A row it updates takes those of them c gives.
func preferencesValues[F domain.PageFilters](c app.PreferencesChange[F]) (preferenceColumns, error) {
	p := domain.DefaultPreferences[F]().Apply(c.Patch)
	filters, filtersErr := json.Marshal(p.Filters)
	displayFilters, displayErr := json.Marshal(p.DisplayFilters)
	displayProperties, propertiesErr := json.Marshal(p.DisplayProperties)
	if err := errors.Join(filtersErr, displayErr, propertiesErr); err != nil {
		return preferenceColumns{}, fmt.Errorf("write the display settings on page %s: %w", c.Page, err)
	}
	return preferenceColumns{filters, displayFilters, displayProperties}, nil
}

// foundPreferences are the settings a row read of page holds, found when
// there was one; err is the read's.
func foundPreferences[F domain.PageFilters](filters, displayFilters, displayProperties []byte, err error, what string, page uuid.UUID) (
	domain.Preferences[F], bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Preferences[F]{}, false, nil
	case err != nil:
		return domain.Preferences[F]{}, false, fmt.Errorf("read the display settings on %s %s: %w", what, page, err)
	}
	p, err := preferencesOf[F](filters, displayFilters, displayProperties)
	if err != nil {
		return domain.Preferences[F]{}, false, err
	}
	return p, true, nil
}

// preferencesOf are the settings a row's three jsonb columns hold.
func preferencesOf[F domain.PageFilters](filters, displayFilters, displayProperties []byte) (domain.Preferences[F], error) {
	var p domain.Preferences[F]
	for _, field := range []struct {
		name string
		data []byte
		into any
	}{
		{"filters", filters, &p.Filters}, {"display_filters", displayFilters, &p.DisplayFilters}, {"display_properties", displayProperties, &p.DisplayProperties},
	} {
		if err := json.Unmarshal(field.data, field.into); err != nil {
			return domain.Preferences[F]{}, fmt.Errorf("read the %s of a page's display settings: %w", field.name, err)
		}
	}
	return p, nil
}
