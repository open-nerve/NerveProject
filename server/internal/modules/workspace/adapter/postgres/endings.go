package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// The statements of an ended membership (M3 design 3.6, 3.7, 3.8):
// removeWorkspaceMember's and leaveWorkspace's, each under the workspace's
// FOR NO KEY UPDATE, in the transaction ctx carries.

// HasOtherAdmin reports whether the workspace has an active admin other
// than userID.
func (s *Store) HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	other, err := s.queries(ctx).HasOtherAdmin(ctx, gen.HasOtherAdminParams{WorkspaceID: workspaceID, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for another admin: %w", err)
	}
	return other, nil
}

// DeletePendingInvitations soft-deletes the workspace's pending invitation
// to email, if there is one, by the account by at now; a declined one
// stays.
func (s *Store) DeletePendingInvitations(ctx context.Context, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).DeletePendingInvitations(ctx, gen.DeletePendingInvitationsParams{WorkspaceID: workspaceID, Email: email, DeletedBy: by,
		Now: now})
	if err != nil {
		return fmt.Errorf("delete the pending invitations: %w", err)
	}
	return nil
}

// EndMember ends userID's membership of the workspace, by the account by at
// now; the row stays. The caller read the membership active under the
// workspace's lock: a pair without exactly one undeleted membership is an
// error, not app.ErrNotFound.
func (s *Store) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	ended, err := s.queries(ctx).EndMember(ctx, gen.EndMemberParams{WorkspaceID: workspaceID, MemberID: userID, EndedBy: by, Now: now})
	switch {
	case err != nil:
		return fmt.Errorf("end workspace member: %w", err)
	case ended != 1:
		return fmt.Errorf("end workspace member %s of %s: %d undeleted memberships", userID, workspaceID, ended)
	}
	return nil
}
