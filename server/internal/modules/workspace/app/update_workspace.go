package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type UpdateWorkspaceRows interface {
	WorkspaceUpdater
	WorkspaceFinder
}

type UpdateWorkspaceDeps struct {
	Workspaces UpdateWorkspaceRows
	Auth       shared.Authorizer
	Caller     CallerLock
	Bindings   ImageBindings
	Images     ImageReader
	Tx         shared.TxManager
	Clock      Clock
}

// UpdateWorkspace changes metadata and the logo in one transaction, with
// account first, then workspace, then all current and next asset rows.
type UpdateWorkspace struct{ d UpdateWorkspaceDeps }

func NewUpdateWorkspace(d UpdateWorkspaceDeps) *UpdateWorkspace { return &UpdateWorkspace{d: d} }

func (u *UpdateWorkspace) Execute(ctx context.Context, slug string, p domain.WorkspacePatch) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	if err := domain.CheckWorkspacePatch(p); err != nil {
		return domain.Workspace{}, err
	}
	var out domain.Workspace
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		id, grant, err := lockAndDecide(ctx, u.d.Workspaces.LockWorkspaceBySlug, u.d.Auth, actor, slug, domain.ActionUpdate)
		if err != nil {
			return err
		}
		current, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			return err
		}
		q := LogoImageChange{ActorID: actor.UserID, WorkspaceID: id, Current: current.LogoAssetID, Next: current.LogoAssetID}
		if p.SetLogo {
			q.Next = p.LogoAssetID
		}
		at, err := u.d.Bindings.LockImageBindings(ctx, q)
		if err != nil {
			return err
		}
		updated, err := u.d.Workspaces.UpdateWorkspace(ctx, id, p, actor.UserID, at)
		if err != nil {
			return err
		}
		if err := u.d.Bindings.ApplyImageBindings(ctx, q, at); err != nil {
			return err
		}
		updated.Role = grant.WorkspaceRole
		list, err := withWorkspaceImages(ctx, u.d.Images, []domain.Workspace{updated})
		if err != nil {
			return err
		}
		out = list[0]
		return nil
	})
	if err != nil {
		return domain.Workspace{}, err
	}
	return out, nil
}
