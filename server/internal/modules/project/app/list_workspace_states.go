package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceStates lists the states of a workspace's projects that the
// caller is a member of: GET /api/v0/workspaces/{slug}/states (M3 design
// 3.4, 3.17).
type ListWorkspaceStates struct {
	workspaces WorkspaceDirectory
	states     WorkspaceStateLister
	auth       shared.Authorizer
}

// NewListWorkspaceStates returns the use case.
func NewListWorkspaceStates(workspaces WorkspaceDirectory, states WorkspaceStateLister, auth shared.Authorizer) *ListWorkspaceStates {
	return &ListWorkspaceStates{workspaces: workspaces, states: states, auth: auth}
}

// Execute finds the workspace slug names, decides workspace_state.list in
// it, then lists the states but the triage states of its unarchived
// projects that the caller is an active member of: those whose states
// state.list lets him list, which no visibility widens (Plane
// views/workspace/state.py:20-26). A workspace not there, or not visible to
// the caller, is domain.ErrWorkspaceNotFound. A read opens no transaction.
func (u *ListWorkspaceStates) Execute(ctx context.Context, slug string) ([]domain.State, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	ws, _, err := findWorkspaceAndDecide(ctx, u.workspaces, u.auth, actor, slug, domain.ActionWorkspaceStateList)
	if err != nil {
		return nil, err
	}
	return u.states.ListWorkspaceStates(ctx, ws.ID, actor.UserID)
}
