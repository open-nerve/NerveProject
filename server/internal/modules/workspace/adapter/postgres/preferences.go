package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// Preferences returns userID's settings in workspaceID; found is false while
// there is no undeleted row.
func (s *Store) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	r, err := s.queries(ctx).Preferences(ctx, gen.PreferencesParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Preferences{}, false, nil
	case err != nil:
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", err)
	}
	return domain.Preferences{NavigationControl: r.NavigationControlPreference, NavigationProjectLimit: int(r.NavigationProjectLimit)}, true, nil
}

// UpsertPreferences applies r.Patch to the account's undeleted row, or
// inserts one with domain.DefaultPreferences and the patch applied, by the
// account at r.Now, and returns the settings as stored. The domain checked
// the values, so the limit fits the column.
func (s *Store) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	inserted := domain.DefaultPreferences().Apply(r.Patch)
	row, err := s.queries(ctx).UpsertPreferences(ctx, gen.UpsertPreferencesParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, UserID: r.UserID,
		NavigationControlPreference: inserted.NavigationControl, NavigationProjectLimit: int32(inserted.NavigationProjectLimit),
		SetNavigationControl: r.Patch.NavigationControl != nil, SetNavigationProjectLimit: r.Patch.NavigationProjectLimit != nil,
		Now: r.Now,
	})
	if err != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", err)
	}
	return domain.Preferences{NavigationControl: row.NavigationControlPreference, NavigationProjectLimit: int(row.NavigationProjectLimit)}, nil
}
