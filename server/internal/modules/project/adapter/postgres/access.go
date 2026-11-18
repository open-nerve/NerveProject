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
	f = app.AccessFacts{WorkspaceID: r.WorkspaceID, Public: domain.Network(r.Network) == domain.NetworkPublic,
		GuestViewAll: r.GuestViewAllFeatures}
	if r.MemberRole != nil {
		f.Member, f.Role = true, shared.Role(*r.MemberRole)
	}
	return f, true, nil
}

// ProjectFactsMany is ProjectFacts for each of ids at once, what
// project.Provide offers the access module's DecideMany (M7 P1b spec 2.7):
// the facts of the undeleted project projectID, archived or not, and the
// project role of each of ids that is its active member; found is false for
// a project that does not exist or is deleted. It reads in the transaction
// ctx carries.
func (s *Store) ProjectFactsMany(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (f app.AccessFactsMany, found bool, err error) {
	rows, err := s.queries(ctx).ProjectFactsMany(ctx, gen.ProjectFactsManyParams{ProjectID: projectID, Ids: ids})
	if err != nil {
		return app.AccessFactsMany{}, false, fmt.Errorf("read project %s for the decisions of %d accounts: %w", projectID, len(ids), err)
	}
	if len(rows) == 0 {
		return app.AccessFactsMany{}, false, nil
	}
	f = app.AccessFactsMany{WorkspaceID: rows[0].WorkspaceID, Public: domain.Network(rows[0].Network) == domain.NetworkPublic,
		GuestViewAll: rows[0].GuestViewAllFeatures, Roles: make(map[uuid.UUID]shared.Role, len(rows))}
	for _, r := range rows {
		if r.MemberID != nil {
			f.Roles[*r.MemberID] = shared.Role(*r.MemberRole)
		}
	}
	return f, true, nil
}

// MemberProjectFacts is ProjectFacts for each project of workspaceID that
// userID is an active member of, what project.Provide offers the access
// module's ReadableProjects (M7 P1c spec 2.5): the undeleted projects,
// archived or not, of his active memberships, each with his project role,
// in id order. It reads in the transaction ctx carries.
func (s *Store) MemberProjectFacts(ctx context.Context, workspaceID, userID uuid.UUID) ([]app.MemberAccessFacts, error) {
	rows, err := s.queries(ctx).MemberProjectFacts(ctx, gen.MemberProjectFactsParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, fmt.Errorf("read the projects of workspace %s that %s is a member of, for a decision: %w", workspaceID, userID, err)
	}
	facts := make([]app.MemberAccessFacts, len(rows))
	for i, r := range rows {
		facts[i] = app.MemberAccessFacts{ProjectID: r.ID, Public: domain.Network(r.Network) == domain.NetworkPublic,
			Role: shared.Role(r.MemberRole), GuestViewAll: r.GuestViewAllFeatures}
	}
	return facts, nil
}
