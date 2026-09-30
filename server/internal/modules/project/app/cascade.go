package app

import (
	"context"
	"time"
	"uuid"
)

// Cascade is what the workspace module's writes ask of the projects,
// workspace's ProjectCascade port (M3 design 3.3). Each method uses
// project's own repositories only, in the transaction ctx carries, which
// the caller began and holds its workspace's lock in: a failure comes back
// as itself and the caller's whole transaction rolls back. The caller gives
// who acts and the moment, which each method writes into the rows it
// changes.
type Cascade struct {
	projects WorkspaceProjectsDeleter
}

// NewCascade returns the cascade over projects.
func NewCascade(projects WorkspaceProjectsDeleter) *Cascade {
	return &Cascade{projects: projects}
}

// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
// rows under them (M3 design 3.6): the last step of deleteWorkspace's
// cascade, under its FOR NO KEY UPDATE of the workspace, each step one
// statement (convention 5). Every step runs, in order, at by and now.
func (c *Cascade) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	for _, step := range c.deletion() {
		if err := step(ctx, workspaceID, by, now); err != nil {
			return err
		}
	}
	return nil
}

// deletion is what deleting a workspace deletes of the projects, in the
// global order of M3 design 3.6: the projects, their memberships, the
// members' display settings, the states. P7 adds the labels at the end.
func (c *Cascade) deletion() []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return []func(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error{
		c.projects.DeleteWorkspaceProjects,
		c.projects.DeleteWorkspaceProjectMembers,
		c.projects.DeleteWorkspaceProjectPreferences,
		c.projects.DeleteWorkspaceStates,
	}
}
