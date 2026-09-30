package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The writes of the answers to an invitation (M3 design 3.8), each under
// the locks the use case holds.

// MemberOf returns userID's undeleted membership of the workspace, active
// or ended; found is false when he has none.
func (s *Store) MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Membership, bool, error) {
	r, err := s.queries(ctx).MemberOf(ctx, gen.MemberOfParams{WorkspaceID: workspaceID, MemberID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Membership{}, false, nil
	case err != nil:
		return domain.Membership{}, false, fmt.Errorf("read workspace member: %w", err)
	}
	return domain.Membership{ID: r.ID, WorkspaceID: r.WorkspaceID, MemberID: r.MemberID, Role: shared.Role(r.Role),
		IsActive: r.IsActive, CreatedAt: r.CreatedAt}, true, nil
}

// RestoreMember makes the membership id active again with role, by the
// account by at now.
func (s *Store) RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).RestoreMember(ctx, gen.RestoreMemberParams{ID: id, Role: int16(role), RestoredBy: &by, Now: now}); err != nil {
		return fmt.Errorf("restore workspace member: %w", err)
	}
	return nil
}

// AcceptInvitation records the invitation id as accepted by the account by
// at now, and deletes it.
func (s *Store) AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).AcceptInvitation(ctx, gen.AcceptInvitationParams{ID: id, AcceptedBy: by, Now: now}); err != nil {
		return fmt.Errorf("accept workspace invitation: %w", err)
	}
	return nil
}

// DeclineInvitation records the invitation id as declined by the account by
// at now.
func (s *Store) DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeclineInvitation(ctx, gen.DeclineInvitationParams{ID: id, DeclinedBy: by, Now: now}); err != nil {
		return fmt.Errorf("decline workspace invitation: %w", err)
	}
	return nil
}

// WorkspaceByID returns the undeleted workspace id and its number of active
// members, without a role. The caller holds the workspace's lock: its
// absence is an error, not app.ErrNotFound.
func (s *Store) WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error) {
	r, err := s.queries(ctx).WorkspaceByID(ctx, id)
	if err != nil {
		return domain.Workspace{}, fmt.Errorf("read workspace: %w", err)
	}
	return domain.Workspace{
		ID: r.ID, Name: r.Name, Slug: r.Slug, OrganizationSize: r.OrganizationSize, Timezone: r.Timezone,
		TotalMembers: int(r.TotalMembers), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}, nil
}
