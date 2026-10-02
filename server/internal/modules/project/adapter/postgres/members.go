package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Memberships are userIDs' undeleted memberships of projectID, active or
// ended, by account (app.MembershipReader).
func (s *Store) Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]app.Membership, error) {
	rows, err := s.queries(ctx).Memberships(ctx, gen.MembershipsParams{ProjectID: projectID, MemberIds: userIDs})
	if err != nil {
		return nil, fmt.Errorf("read memberships of project %s: %w", projectID, err)
	}
	out := make(map[uuid.UUID]app.Membership, len(rows))
	for _, r := range rows {
		out[r.MemberID] = app.Membership{ID: r.ID, Role: shared.Role(r.Role), Active: r.IsActive}
	}
	return out, nil
}

// ListMembers lists projectID's active memberships, in the order they were
// made (app.MemberLister).
func (s *Store) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	rows, err := s.queries(ctx).ListMembers(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list members of project %s: %w", projectID, err)
	}
	out := make([]domain.Member, len(rows))
	for i, r := range rows {
		out[i] = domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}
	}
	return out, nil
}
