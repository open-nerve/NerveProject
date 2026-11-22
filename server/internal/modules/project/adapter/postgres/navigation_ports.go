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
)

// NavigationFacts reads the undeleted project id's facts for the navigation
// module without a lock (navigation_ports.sql); found is false when there is
// none.
func (s *Store) NavigationFacts(ctx context.Context, id uuid.UUID) (app.NavigationFacts, bool, error) {
	r, err := s.queries(ctx).NavigationProjectFacts(ctx, id)
	return navigationFacts(gen.ShareNavigationProjectRow(r), err, "read the navigation facts of project", id)
}

// ShareNavigation locks the undeleted project id FOR SHARE, its workspace
// not locked, and reads its facts; found is false when there is none, also
// when it was deleted while the lock waited.
func (s *Store) ShareNavigation(ctx context.Context, id uuid.UUID) (app.NavigationFacts, bool, error) {
	r, err := s.queries(ctx).ShareNavigationProject(ctx, id)
	return navigationFacts(r, err, "share project for a view's write", id)
}

func navigationFacts(r gen.ShareNavigationProjectRow, err error, what string, id uuid.UUID) (app.NavigationFacts, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.NavigationFacts{}, false, nil
	case err != nil:
		return app.NavigationFacts{}, false, fmt.Errorf("%s %s: %w", what, id, err)
	}
	return app.NavigationFacts{WorkspaceID: r.WorkspaceID, IssueViewsView: r.IssueViewsView}, true, nil
}

// NavigationTargets reads, in one statement and without a lock, the
// undeleted projects among ids, archived or not, of any workspace, by
// project (navigation_ports.sql): what project.Provide offers the
// navigation module's favorites and recent visits (M7 P3c spec 2.4, 2.5,
// P3d spec 2.4). A deleted project and an id of none are not among them.
// It reads in the transaction ctx carries or on the pool.
func (s *Store) NavigationTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.NavigationTarget, error) {
	rows, err := s.queries(ctx).NavigationTargets(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d projects for the navigation module: %w", len(ids), err)
	}
	targets := make(map[uuid.UUID]app.NavigationTarget, len(rows))
	for _, r := range rows {
		t := app.NavigationTarget{WorkspaceID: r.WorkspaceID, Name: r.Name, Identifier: r.Identifier, Archived: r.Archived}
		if err := json.Unmarshal(r.LogoProps, &t.LogoProps); err != nil {
			return nil, fmt.Errorf("read logo_props of project %s: %w", r.ID, err)
		}
		targets[r.ID] = t
	}
	return targets, nil
}
