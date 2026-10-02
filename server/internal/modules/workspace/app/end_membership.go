package app

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

// membershipEnd is the step removeWorkspaceMember and leaveWorkspace share
// (M3 design 3.6's lock table, 3.7 rule 2, 3.8): it ends an account's
// membership of a workspace, under the caller's FOR NO KEY UPDATE of it, at
// the caller's moment, read under that lock, by the caller's account.
type membershipEnd struct {
	members  MembershipEnder
	profiles MemberProfiles
	projects ProjectCascade
}

// run ends userID's membership of the workspace, in the caller's
// transaction: his address, read through MemberProfiles without a lock
// (convention 1); the workspace's pending invitation to it soft-deleted, so
// that the ended membership leaves no invitation (3.8), before his
// membership's row, as the global order has the invitations before the
// members; his membership ended, the row kept; then his memberships of the
// workspace's projects ended (ProjectCascade.EndMemberships), which refuses
// with project.sole_admin when he is the only active admin of a project
// with other active members (3.7 rule 2). A failure comes back as itself
// and nothing runs after it; the caller's transaction rolls back, the
// invitation's deletion with it.
func (e membershipEnd) run(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	profiles, err := e.profiles.PublicProfiles(ctx, []uuid.UUID{userID})
	if err != nil {
		return err
	}
	if len(profiles) != 1 || profiles[0].ID != userID {
		// The foreign key keeps every member's account: its absence is a bug,
		// and another account's address would delete its invitations.
		return fmt.Errorf("end the membership of %s in %s: no account", userID, workspaceID)
	}
	if err := e.members.DeletePendingInvitations(ctx, workspaceID, profiles[0].Email, by, now); err != nil {
		return err
	}
	if err := e.members.EndMember(ctx, workspaceID, userID, by, now); err != nil {
		return err
	}
	return e.projects.EndMemberships(ctx, []uuid.UUID{workspaceID}, userID, by, now)
}
