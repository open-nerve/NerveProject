package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// MemberByID is the undeleted project membership id, active or ended;
// found is false when there is none (app.MemberFinder).
func (s *Store) MemberByID(ctx context.Context, id uuid.UUID) (m app.ProjectMembership, found bool, err error) {
	r, err := s.queries(ctx).MemberByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.ProjectMembership{}, false, nil
	case err != nil:
		return app.ProjectMembership{}, false, fmt.Errorf("read project membership %s: %w", id, err)
	}
	return app.ProjectMembership{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		Active: r.IsActive}, true, nil
}

// UpdateMemberRole gives the active membership id role, by the account by
// at now, and returns it as stored (app.MemberRoleChanger). An ended or
// deleted one is an error, not written.
func (s *Store) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Member, error) {
	r, err := s.queries(ctx).UpdateMemberRole(ctx, gen.UpdateMemberRoleParams{ID: id, Role: int16(role), UpdatedBy: by, Now: now})
	if err != nil {
		return domain.Member{}, fmt.Errorf("change the role of project membership %s: %w", id, err)
	}
	return domain.Member{ID: r.ID, ProjectID: r.ProjectID, MemberID: r.MemberID, Role: shared.Role(r.Role), CreatedAt: r.CreatedAt}, nil
}

// EndMember ends userID's active membership of projectID, by the account
// by at now (app.MemberEnder): anything but exactly one row ended is an
// error.
func (s *Store) EndMember(ctx context.Context, projectID, userID, by uuid.UUID, now time.Time) error {
	n, err := s.queries(ctx).EndMember(ctx, gen.EndMemberParams{ProjectID: projectID, MemberID: userID, EndedBy: by, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("end the membership of %s of project %s: %w", userID, projectID, err)
	case n != 1:
		return fmt.Errorf("end the membership of %s of project %s: %d rows ended, want 1", userID, projectID, n)
	}
	return nil
}

// HasOtherAdmin reports whether projectID has an active admin other than
// userID (app.MemberLeaver).
func (s *Store) HasOtherAdmin(ctx context.Context, projectID, userID uuid.UUID) (bool, error) {
	other, err := s.queries(ctx).HasOtherAdmin(ctx, gen.HasOtherAdminParams{ProjectID: projectID, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for another admin of project %s: %w", projectID, err)
	}
	return other, nil
}
