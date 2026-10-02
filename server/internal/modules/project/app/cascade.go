package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// Cascade is what the workspace module's writes ask of the projects,
// workspace's ProjectCascade port (M3 design 3.3). Each method uses
// project's own repositories only, in the transaction ctx carries, which
// the caller began and holds its workspace's lock in: a failure comes back
// as itself and the caller's whole transaction rolls back. The caller gives
// who acts and the moment, which each method writes into the rows it
// changes.
type Cascade struct {
	projects ProjectsDeleter
	members  MemberDemoter
	enders   MembershipEnder
}

// NewCascade returns the cascade over projects, members and enders.
func NewCascade(projects ProjectsDeleter, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, members: members, enders: enders}
}

// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
// rows under them (M3 design 3.6): the last step of deleteWorkspace's
// cascade, under its FOR NO KEY UPDATE of the workspace, each step one
// statement (convention 5), the steps deleteProject runs too
// (deleteProjects), at by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return deleteProjects(ctx, c.projects, Deletion{WorkspaceID: workspaceID, By: by, Now: now})
}

// DemoteToGuest makes userID a guest in each of the workspace's projects he
// has a membership of, ended ones too (M3 design 3.3; Plane
// views/workspace/member.py:87-89): under the caller's FOR NO KEY UPDATE of
// the workspace, those projects FOR NO KEY UPDATE in id order, then his
// memberships of them in one statement (convention 5), at now, by by. With
// no such project there is nothing to write.
func (c *Cascade) DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	projects, err := c.members.LockMemberProjects(ctx, workspaceID, userID)
	if err != nil || len(projects) == 0 {
		return err
	}
	return c.members.DemoteMemberships(ctx, projects, userID, by, now)
}

// EndMemberships ends userID's active memberships of the workspaces'
// projects (M3 design 3.6 convention 6, 3.7 rule 2), under the caller's FOR
// NO KEY UPDATE of each workspace, once the caller has ended his membership
// of it: the projects, found now, not given (a growth that committed before
// the caller's lock is among them), locked FOR NO KEY UPDATE in id order;
// then, were he the only active admin of one that has another active
// member, domain.ErrSoleAdmin, nothing written; else his memberships of
// them ended in one statement (convention 5), at now, by by. With no such
// project there is nothing to write.
func (c *Cascade) EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	projects, err := c.enders.LockActiveMemberProjects(ctx, workspaceIDs, userID)
	if err != nil || len(projects) == 0 {
		return err
	}
	sole, err := c.enders.SoleAdmin(ctx, projects, userID)
	switch {
	case err != nil:
		return err
	case sole:
		return domain.ErrSoleAdmin
	}
	return c.enders.EndMemberships(ctx, projects, userID, by, now)
}
