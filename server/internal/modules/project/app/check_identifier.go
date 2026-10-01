package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CheckProjectIdentifier answers whether an identifier is available in a
// workspace: GET /api/v0/workspaces/{slug}/project-identifiers/{identifier}
// (M3 design 5.1).
type CheckProjectIdentifier struct {
	workspaces WorkspaceDirectory
	projects   IdentifierReader
	auth       shared.Authorizer
}

// NewCheckProjectIdentifier returns the use case.
func NewCheckProjectIdentifier(workspaces WorkspaceDirectory, projects IdentifierReader, auth shared.Authorizer) *CheckProjectIdentifier {
	return &CheckProjectIdentifier{workspaces: workspaces, projects: projects, auth: auth}
}

// Execute finds the workspace slug, decides project_identifier.check in it,
// then answers whether createProject would take identifier: valid once
// upper-cased (domain.ValidIdentifier), and no undeleted project of the
// workspace's. A workspace not there, or not visible to the caller, is
// domain.ErrWorkspaceNotFound. A read opens no transaction (M3 design 6.7).
func (u *CheckProjectIdentifier) Execute(ctx context.Context, slug, identifier string) (bool, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return false, err
	}
	ws, found, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case err != nil:
		return false, err
	case !found:
		return false, domain.ErrWorkspaceNotFound
	}
	_, err = u.auth.Authorize(ctx, actor, domain.ActionCheckIdentifier, shared.Target{WorkspaceID: ws.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return false, domain.ErrWorkspaceNotFound
	case err != nil:
		return false, err
	}
	if !domain.ValidIdentifier(identifier) {
		return false, nil
	}
	taken, err := u.projects.IdentifierTaken(ctx, ws.ID, domain.Identifier(identifier))
	if err != nil {
		return false, err
	}
	return !taken, nil
}
