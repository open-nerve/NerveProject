package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListWorkspaceLabels lists the labels of a workspace's projects that the
// caller is a member of: GET /api/v0/workspaces/{slug}/labels (M7/P6b spec
// 2.1).
type ListWorkspaceLabels struct {
	workspaces WorkspaceDirectory
	labels     WorkspaceLabelLister
	auth       shared.Authorizer
}

// NewListWorkspaceLabels returns the use case.
func NewListWorkspaceLabels(workspaces WorkspaceDirectory, labels WorkspaceLabelLister, auth shared.Authorizer) *ListWorkspaceLabels {
	return &ListWorkspaceLabels{workspaces: workspaces, labels: labels, auth: auth}
}

// Execute finds the workspace slug names, decides workspace_label.list in
// it, then lists the labels of its unarchived projects that the caller is
// an active member of, parents and children alike: those whose labels
// label.list lets him list, which no visibility widens, but the archived
// projects', which listLabels lists (Plane views/workspace/label.py:17-30).
// A workspace not there, or not visible to the caller, is
// domain.ErrWorkspaceNotFound. A read opens no transaction.
func (u *ListWorkspaceLabels) Execute(ctx context.Context, slug string) ([]domain.Label, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	ws, _, err := findWorkspaceAndDecide(ctx, u.workspaces, u.auth, actor, slug, domain.ActionWorkspaceLabelList)
	if err != nil {
		return nil, err
	}
	return u.labels.ListWorkspaceLabels(ctx, ws.ID, actor.UserID)
}
