package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetWorkspace reads a workspace by its slug: GET /api/v0/workspaces/{slug}.
type GetWorkspace struct {
	workspaces WorkspaceFinder
	auth       shared.Authorizer
}

// NewGetWorkspace returns the use case.
func NewGetWorkspace(workspaces WorkspaceFinder, auth shared.Authorizer) *GetWorkspace {
	return &GetWorkspace{workspaces: workspaces, auth: auth}
}

// Execute returns the workspace with the caller's role. A read opens no
// transaction and decides directly (M3 design 3.4). A workspace that does
// not exist, is deleted, or that the caller cannot see is the same
// workspace.not_found (M3 design 8.2).
func (u *GetWorkspace) Execute(ctx context.Context, slug string) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	w, err := u.workspaces.WorkspaceBySlug(ctx, slug)
	switch {
	case errors.Is(err, ErrNotFound):
		return domain.Workspace{}, domain.ErrNotFound
	case err != nil:
		return domain.Workspace{}, err
	}
	grant, err := u.auth.Authorize(ctx, actor, domain.ActionRead, shared.Target{WorkspaceID: w.ID})
	switch {
	case errors.Is(err, shared.ErrNotVisible):
		return domain.Workspace{}, domain.ErrNotFound
	case err != nil:
		return domain.Workspace{}, err
	}
	w.Role = grant.WorkspaceRole
	return w, nil
}
