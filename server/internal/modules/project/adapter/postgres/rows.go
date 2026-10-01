package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// The rows under a project: its memberships, its members' display settings
// and its states, one statement a row.

// CreateMember inserts m, active.
func (s *Store) CreateMember(ctx context.Context, m app.MemberRow) error {
	err := s.queries(ctx).CreateMember(ctx, gen.CreateMemberParams{
		ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, MemberID: m.MemberID, Role: int16(m.Role),
		CreatedBy: &m.CreatedBy, Now: m.Now,
	})
	if err != nil {
		return fmt.Errorf("create project member: %w", err)
	}
	return nil
}

// CreatePreferences inserts p, the navigation at its default.
func (s *Store) CreatePreferences(ctx context.Context, p app.PreferencesRow) error {
	err := s.queries(ctx).CreatePreferences(ctx, gen.CreatePreferencesParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, ProjectID: p.ProjectID, UserID: p.UserID, SortOrder: p.SortOrder,
		CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	if err != nil {
		return fmt.Errorf("create project preferences: %w", err)
	}
	return nil
}

// LowestSortOrder returns the least place of userID's in his sidebar among
// workspaceID's projects, over his undeleted display settings, those of
// projects he left too; nil when he has none (M3 design 3.18).
func (s *Store) LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error) {
	lowest, err := s.queries(ctx).LowestSortOrder(ctx, gen.LowestSortOrderParams{WorkspaceID: workspaceID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the lowest sort order: %w", err)
	}
	return &lowest, nil
}

// CreateStates inserts rows in the order given, and stops at the first
// that fails.
func (s *Store) CreateStates(ctx context.Context, rows []app.StateRow) error {
	for _, r := range rows {
		err := s.queries(ctx).CreateState(ctx, gen.CreateStateParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Name: r.State.Name, Color: r.State.Color,
			Sequence: r.State.Sequence, StateGroup: string(r.State.Group), IsDefault: r.State.Default, CreatedBy: &r.CreatedBy, Now: r.Now,
		})
		if err != nil {
			return fmt.Errorf("create state %q: %w", r.State.Name, err)
		}
	}
	return nil
}
