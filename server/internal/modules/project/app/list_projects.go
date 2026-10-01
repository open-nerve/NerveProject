package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListProjects lists a workspace's projects that the caller sees: GET
// /api/v0/workspaces/{slug}/projects (M3 design 3.4, 3.12, 3.19).
type ListProjects struct {
	workspaces WorkspaceDirectory
	projects   ProjectLister
	auth       shared.Authorizer
}

// NewListProjects returns the use case.
func NewListProjects(workspaces WorkspaceDirectory, projects ProjectLister, auth shared.Authorizer) *ListProjects {
	return &ListProjects{workspaces: workspaces, projects: projects, auth: auth}
}

// Execute finds the workspace slug, decides project.list in it, then lists
// the projects that the caller's workspace role, as the decision read it,
// sees (domain.VisibilityOf) besides those he is a member of: the archived
// ones alone when archived is true, the others otherwise. A workspace not
// there, or not visible to the caller, is domain.ErrWorkspaceNotFound. A
// read opens no transaction (M3 design 6.7).
func (u *ListProjects) Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	ws, found, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return nil, err
	case !found:
		return nil, domain.ErrWorkspaceNotFound
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionList, shared.Target{WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return nil, domain.ErrWorkspaceNotFound
	case err != nil:
		return nil, err
	}
	return u.projects.ListProjects(ctx, ws.ID, actor.UserID, domain.VisibilityOf(grant.WorkspaceRole), archived)
}
