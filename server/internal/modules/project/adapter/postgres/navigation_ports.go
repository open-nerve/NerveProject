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
