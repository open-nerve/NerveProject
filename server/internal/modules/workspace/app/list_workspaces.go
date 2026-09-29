package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaces lists the caller's workspaces: GET /api/v0/workspaces, the
// whole collection at once (M3 design 3.12). It is an account-level
// operation: any valid credential may call it, and the list holds only the
// workspaces of which the caller is an active member, so it asks no
// Authorizer (M3 design 6.4).
type ListWorkspaces struct {
	workspaces WorkspaceLister
}

// NewListWorkspaces returns the use case.
func NewListWorkspaces(workspaces WorkspaceLister) *ListWorkspaces {
	return &ListWorkspaces{workspaces: workspaces}
}

// Execute returns the caller's workspaces, by name then id.
func (u *ListWorkspaces) Execute(ctx context.Context) ([]domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	return u.workspaces.ListWorkspaces(ctx, actor.UserID)
}
