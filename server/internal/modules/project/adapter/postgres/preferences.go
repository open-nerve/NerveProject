package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// navigationJSON is project_user_properties.preferences as stored: one key,
// navigation, with exactly its two (M3 design 4.8).
type navigationJSON struct {
	Navigation struct {
		DefaultTab     string   `json:"default_tab"`
		HideInMoreMenu []string `json:"hide_in_more_menu"`
	} `json:"navigation"`
}

// Preferences returns userID's display settings in projectID; found is
// false while he has no undeleted row of them (app.PreferencesReader).
func (s *Store) Preferences(ctx context.Context, projectID, userID uuid.UUID) (domain.Preferences, bool, error) {
	r, err := s.queries(ctx).Preferences(ctx, gen.PreferencesParams{ProjectID: projectID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Preferences{}, false, nil
	case err != nil:
		return domain.Preferences{}, false, fmt.Errorf("read project preferences: %w", err)
	}
	p, err := preferences(r.Preferences, r.SortOrder)
	if err != nil {
		return domain.Preferences{}, false, err
	}
	return p, true, nil
}

// UpsertPreferences applies c.Patch to the account's undeleted row, or
// inserts one with domain.DefaultPreferences and the patch applied, by the
// account at c.Now, and returns the settings as stored
// (app.PreferencesWriter).
func (s *Store) UpsertPreferences(ctx context.Context, c app.PreferencesChange) (domain.Preferences, error) {
	inserted := domain.DefaultPreferences().Apply(c.Patch)
	var n navigationJSON
	n.Navigation.DefaultTab, n.Navigation.HideInMoreMenu = inserted.Navigation.DefaultTab, inserted.Navigation.HideInMoreMenu
	if n.Navigation.HideInMoreMenu == nil {
		n.Navigation.HideInMoreMenu = []string{}
	}
	stored, err := json.Marshal(n)
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write project preferences: %w", err)
	}
	r, err := s.queries(ctx).UpsertPreferences(ctx, gen.UpsertPreferencesParams{
		ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, UserID: c.UserID, Preferences: stored, SortOrder: inserted.SortOrder,
		SetNavigation: c.Patch.Navigation != nil, SetSortOrder: c.Patch.SortOrder != nil, Now: c.Now,
	})
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write project preferences: %w", err)
	}
	return preferences(r.Preferences, r.SortOrder)
}

// EnsurePreferences inserts p unless its account has undeleted display
// settings in its project already, which stay as they are
// (app.MemberGrower).
func (s *Store) EnsurePreferences(ctx context.Context, p app.PreferencesRow) error {
	err := s.queries(ctx).EnsurePreferences(ctx, gen.EnsurePreferencesParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, ProjectID: p.ProjectID, UserID: p.UserID, SortOrder: p.SortOrder, CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	if err != nil {
		return fmt.Errorf("ensure project preferences: %w", err)
	}
	return nil
}

// preferences are the settings a row holds.
func preferences(stored []byte, sortOrder float64) (domain.Preferences, error) {
	var n navigationJSON
	if err := json.Unmarshal(stored, &n); err != nil {
		return domain.Preferences{}, fmt.Errorf("read project preferences %s: %w", stored, err)
	}
	return domain.Preferences{
		Navigation: domain.Navigation{DefaultTab: n.Navigation.DefaultTab, HideInMoreMenu: n.Navigation.HideInMoreMenu},
		SortOrder:  sortOrder,
	}, nil
}
