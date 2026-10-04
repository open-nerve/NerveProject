package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// Deactivator ends a deactivated account's memberships: identity's
// MembershipDeactivator (M3 design 3.9), which bootstrap wires.
type Deactivator struct {
	memberships AllMembershipsEnder
	projects    ProjectCascade
	clock       Clock
}

// NewDeactivator returns the Deactivator over memberships and projects,
// the project module's cascade.
func NewDeactivator(memberships AllMembershipsEnder, projects ProjectCascade, clock Clock) *Deactivator {
	return &Deactivator{memberships: memberships, projects: projects, clock: clock}
}

// DeactivateMemberships ends every membership of userID, whose address
// is email as read under his account row's lock, in the transaction ctx
// carries: identity's deactivation began it, holds that row FOR NO KEY
// UPDATE, and calls this last (M3 design 3.6 convention 6, 3.9). It locks
// the undeleted workspaces of which he is an active member FOR NO KEY
// UPDATE in id order, found now: no growth of his set of workspaces can
// commit once his row is held, and one deleted before its lock is left
// out. Were he the only active admin of one that has another active
// member, domain.ErrSoleAdmin, nothing written (3.7 rule 2). Else it locks
// the invitations it deletes, as found then, FOR NO KEY UPDATE in id
// order: every invitation to email, of any workspace, pending or declined
// (3.8), and the pending invitations of each of those workspaces where he
// is the only active member, so that none lets anyone into a workspace
// left with no member (3.7). Every deactivation takes them so, and writes
// no other invitation, so that two do not wait for each other over them
// (3.6's global order). It reads the clock, once, after every lock so far,
// the last workspace's among them (3.3); and, at that moment and by him:
// those invitations deleted, by the ids the lock returned; his memberships
// of those workspaces ended; then the project cascade's EndMemberships,
// called once across them, which locks his projects in id order and may
// refuse with project.sole_admin. A refusal or failure comes back as
// itself, and identity rolls the whole deactivation back.
func (d *Deactivator) DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error {
	workspaces, err := d.memberships.LockMemberWorkspaces(ctx, userID)
	if err != nil {
		return err
	}
	sole, err := d.memberships.SoleAdmin(ctx, workspaces, userID)
	switch {
	case err != nil:
		return err
	case sole:
		return domain.ErrSoleAdmin
	}
	invitations, err := d.memberships.LockInvitationsToDelete(ctx, workspaces, userID, email)
	if err != nil {
		return err
	}
	now := d.clock.Now()
	if err := d.memberships.DeleteInvitations(ctx, invitations, userID, now); err != nil {
		return err
	}
	if err := d.memberships.EndWorkspaceMemberships(ctx, workspaces, userID, userID, now); err != nil {
		return err
	}
	return d.projects.EndMemberships(ctx, workspaces, userID, userID, now)
}
