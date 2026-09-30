package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
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
		case uniqueViolation(err, "workspace_member_invites_workspace_id_email_key"):
			return nil, &app.DuplicateInvitation{Email: r.Email}
		case err != nil:
			return nil, fmt.Errorf("create workspace invitation: %w", err)
		}
		out = append(out, invitation(row))
	}
	return out, nil
}

// DeleteWorkspaceInvitations soft-deletes the undeleted invitations of the
// workspace, pending or declined, by the account by at now.
func (s *Store) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeleteWorkspaceInvitations(ctx, gen.DeleteWorkspaceInvitationsParams{WorkspaceID: workspaceID, DeletedBy: by, Now: now})
	if err != nil {
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
