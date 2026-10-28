package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// Cascade performs project M3 changes in the caller transaction.
type Cascade struct {
	projects ProjectsDeleter
	locker   WorkspaceProjectsLocker
	members  MemberDemoter
	enders   MembershipEnder
}

// NewCascade returns the cascade over project rows and memberships.
func NewCascade(projects ProjectsDeleter, locker WorkspaceProjectsLocker, members MemberDemoter, enders MembershipEnder) *Cascade {
	return &Cascade{projects: projects, locker: locker, members: members, enders: enders}
}

// LockWorkspaceProjects locks the workspace's undeleted projects, archived
// ones too, FOR NO KEY UPDATE in id order, and returns their ids in that
// order (M4 design 3.5): deleteWorkspace takes it under its FOR NO KEY
// UPDATE of the workspace and after the lock of the invitations, before it
// reads its moment, so that the moment is later than that of every write
// that holds one of these projects.
func (c *Cascade) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	return c.locker.LockWorkspaceProjects(ctx, workspaceID)
}

// DeleteWorkspaceProjects deletes M3 project rows under the caller's workspace
// and project locks, using the request's actor and moment.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, by uuid.UUID, now time.Time) error {
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
