package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateInvitations inserts rows, one statement each, in the order given,
// and returns them as stored, in that order. The first row whose address an
// undeleted invitation of its workspace has is *app.DuplicateInvitation,
// naming that address; nothing after it is inserted, and the caller's
// transaction, aborted, rolls back the rows before it. The domain checked
// every value, so a CHECK violation is a bug: an internal error.
func (s *Store) CreateInvitations(ctx context.Context, rows []app.InvitationRow) ([]domain.Invitation, error) {
	q := s.queries(ctx)
	out := make([]domain.Invitation, 0, len(rows))
	for _, r := range rows {
		row, err := q.CreateInvitation(ctx, gen.CreateInvitationParams{
			ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: int16(r.Role), CreatedBy: &r.CreatedBy, Now: r.Now,
		})
		switch {
		case postgres.UniqueViolation(err, "workspace_member_invites_workspace_id_email_key"):
			return nil, &app.DuplicateInvitation{Email: r.Email}
		case err != nil:
			return nil, fmt.Errorf("create workspace invitation: %w", err)
		}
		out = append(out, invitation(row))
	}
	return out, nil
}

// ListInvitations returns the workspace's undeleted invitations, pending or
// declined, newest first, then by id.
func (s *Store) ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error) {
	rows, err := s.queries(ctx).ListInvitations(ctx, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", err)
	}
	out := make([]domain.Invitation, len(rows))
	for i, r := range rows {
		out[i] = invitation(r)
	}
	return out, nil
}

// InvitationByID returns the undeleted invitation id; app.ErrNotFound when
// there is none.
func (s *Store) InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	r, err := s.queries(ctx).InvitationByID(ctx, id)
	if err != nil {
		return domain.Invitation{}, notFound(err)
	}
	return invitation(r), nil
}

// InvitationPreview returns the undeleted invitation id of an undeleted
// workspace as its link shows it; app.ErrNotFound when there is none.
func (s *Store) InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error) {
	r, err := s.queries(ctx).InvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, notFound(err)
	}
	return domain.InvitationPreview{ID: r.ID, Role: shared.Role(r.Role), Declined: r.RespondedAt != nil, WorkspaceName: r.WorkspaceName,
		WorkspaceSlug: r.WorkspaceSlug}, nil
}

// LockInvitation locks the undeleted invitation id FOR UPDATE until the
// transaction ends and returns it; app.ErrNotFound when there is none, also
// when it was deleted while the lock waited.
func (s *Store) LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error) {
	r, err := s.queries(ctx).LockInvitation(ctx, id)
	if err != nil {
		return domain.Invitation{}, notFound(err)
	}
	return invitation(r), nil
}

// UpdateInvitationRole sets the role of the invitation id, by the account by
// at now, and returns it as stored. The caller holds the invitation's lock:
// its absence is an error.
func (s *Store) UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error) {
	r, err := s.queries(ctx).UpdateInvitationRole(ctx, gen.UpdateInvitationRoleParams{ID: id, Role: int16(role), UpdatedBy: &by, Now: now})
	if err != nil {
		return domain.Invitation{}, fmt.Errorf("update workspace invitation: %w", err)
	}
	return invitation(r), nil
}

// DeleteInvitation soft-deletes the invitation id, by the account by at
// now.
func (s *Store) DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteInvitation(ctx, gen.DeleteInvitationParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete workspace invitation: %w", err)
	}
	return nil
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
// workspace, pending or declined, by the account by at now. It locks them
// FOR NO KEY UPDATE in id order, then deletes exactly those, in the
// caller's transaction (M3 design 3.6 convention 5; ruling G-1).
func (s *Store) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	q := s.queries(ctx)
	ids, err := q.LockWorkspaceInvitations(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("lock workspace invitations: %w", err)
	}
	if err := q.DeleteInvitations(ctx, gen.DeleteInvitationsParams{Ids: ids, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete workspace invitations: %w", err)
	}
	return nil
}

// invitation is a stored row as the domain's value.
func invitation(r gen.WorkspaceMemberInvite) domain.Invitation {
	return domain.Invitation{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Email: r.Email, Role: shared.Role(r.Role), Accepted: r.Accepted,
		RespondedAt: r.RespondedAt, CreatedAt: r.CreatedAt, CreatedByID: r.CreatedByID,
	}
}
