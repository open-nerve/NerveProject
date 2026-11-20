package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// navigationProjects is project's NavigationProjects as navigation's
// Projects (M7 P3b spec 2.1, 2.6): the same read and lock, the facts
// converted.
type navigationProjects struct {
	projects project.NavigationProjects
}

func (p navigationProjects) Facts(ctx context.Context, id uuid.UUID) (navigation.ProjectFacts, bool, error) {
	f, found, err := p.projects.NavigationFacts(ctx, id)
	return navigation.ProjectFacts(f), found, err
}

func (p navigationProjects) Share(ctx context.Context, id uuid.UUID) (navigation.ProjectFacts, bool, error) {
	f, found, err := p.projects.ShareNavigation(ctx, id)
	return navigation.ProjectFacts(f), found, err
}

// navigationWorkspaces is workspace's WorkspaceDirectory as navigation's
// Workspaces: the same read and locks, the workspace's id alone.
type navigationWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d navigationWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

func (d navigationWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, bool, error) {
	w, found, err := d.directory.ShareWorkspaceBySlug(ctx, slug)
	return w.ID, found, err
}

func (d navigationWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (bool, error) {
	_, found, err := d.directory.ShareWorkspaceByID(ctx, id)
	return found, err
}
