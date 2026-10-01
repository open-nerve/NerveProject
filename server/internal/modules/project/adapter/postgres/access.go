package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts is what project.Provide offers the access module (M3 design
// 6.5): the facts of the undeleted project projectID, archived or not, for
// userID; found is false for a project that does not exist or is deleted.
// It reads in the transaction ctx carries.
func (s *Store) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f app.AccessFacts, found bool, err error) {
	r, err := s.queries(ctx).ProjectFacts(ctx, gen.ProjectFactsParams{ProjectID: projectID, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.AccessFacts{}, false, nil
	case err != nil:
		return app.AccessFacts{}, false, fmt.Errorf("read project %s for a decision: %w", projectID, err)
	}
	f = app.AccessFacts{WorkspaceID: r.WorkspaceID, Public: domain.Network(r.Network) == domain.NetworkPublic}
	if r.MemberRole != nil {
		f.Member, f.Role = true, shared.Role(*r.MemberRole)
	}
	return f, true, nil
}
