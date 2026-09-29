package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspace changes a workspace's name, organization size or time
// zone: PATCH /api/v0/workspaces/{slug}.
type UpdateWorkspace struct {
	workspaces WorkspaceUpdater
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
}

// NewUpdateWorkspace returns the use case.
func NewUpdateWorkspace(workspaces WorkspaceUpdater, auth shared.Authorizer, tx shared.TxManager, clock Clock) *UpdateWorkspace {
	return &UpdateWorkspace{workspaces: workspaces, auth: auth, tx: tx, clock: clock}
}

// Execute checks p, then in one transaction (M3 design 3.6): the workspace
// row FOR NO KEY UPDATE, the decision on workspace.update, the change. The
// answer carries the caller's role from the grant. A check of the values
// alone comes first: it tells nothing about the workspace.
func (u *UpdateWorkspace) Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	if err := domain.CheckWorkspacePatch(p); err != nil {
		return domain.Workspace{}, err
	}
	now := u.clock.Now()
	var updated domain.Workspace
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, grant, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionUpdate)
		if err != nil {
			return err
		}
		if updated, err = u.workspaces.UpdateWorkspace(ctx, id, p, actor.UserID, now); err != nil {
			return err
		}
		updated.Role = grant.WorkspaceRole
		return nil
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return updated, nil
}
