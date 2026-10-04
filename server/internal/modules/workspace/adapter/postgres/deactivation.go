package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres/gen"
)

// The statements of a deactivated account's ending (M3 design 3.6
// convention 6, 3.9): app.AllMembershipsEnder, in the transaction ctx
// carries, which identity's deactivation began and holds the account row's
// FOR NO KEY UPDATE in.

// LockMemberWorkspaces locks the undeleted workspaces of which userID is an
// active member FOR NO KEY UPDATE in id order, until the transaction ends,
// and returns their ids in that order. One deleted while the lock waited is
// left out.
func (s *Store) LockMemberWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).LockMemberWorkspaces(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("lock the member's workspaces: %w", err)
	}
	return ids, nil
}

// SoleAdmin reports whether userID is the only active admin of one of the
// workspaces that has another active member.
func (s *Store) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	sole, err := s.queries(ctx).SoleAdmin(ctx, gen.SoleAdminParams{WorkspaceIds: workspaceIDs, MemberID: userID})
	if err != nil {
		return false, fmt.Errorf("look for a workspace he is the only admin of: %w", err)
	}
	return sole, nil
}

// DeleteInvitationsTo soft-deletes every undeleted invitation to email,
// pending or declined, of every workspace, by the account by at now.
func (s *Store) DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteInvitationsTo(ctx, gen.DeleteInvitationsToParams{Email: email, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete the invitations to the address: %w", err)
	}
	return nil
}

// DeleteInvitationsOfWorkspacesLeftEmpty soft-deletes the pending
// invitations of each of the workspaces where userID has no other active
// member, by the account by at now. Call it before his memberships end.
func (s *Store) DeleteInvitationsOfWorkspacesLeftEmpty(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID,
	now time.Time) error {
	err := s.queries(ctx).DeleteInvitationsOfWorkspacesLeftEmpty(ctx, gen.DeleteInvitationsOfWorkspacesLeftEmptyParams{
		WorkspaceIds: workspaceIDs, MemberID: userID, DeletedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("delete the invitations of the workspaces left empty: %w", err)
	}
	return nil
}

// EndWorkspaceMemberships ends userID's active, undeleted memberships of
// the workspaces, by the account by at now; the rows stay. A workspace of
// workspaceIDs where he has none, his membership of it ended while its
// lock waited, is no error.
func (s *Store) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	err := s.queries(ctx).EndWorkspaceMemberships(ctx, gen.EndWorkspaceMembershipsParams{
		WorkspaceIds: workspaceIDs, MemberID: userID, EndedBy: by, Now: now,
	})
	if err != nil {
		return fmt.Errorf("end the workspace memberships: %w", err)
	}
	return nil
}
