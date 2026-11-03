package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ListWorkspaceRows interface {
	WorkspaceLister
	ImageWorkspaceSharer
}

type ListWorkspacesDeps struct {
	Workspaces ListWorkspaceRows
	Caller     CallerLock
	Images     ImageReader
	Tx         shared.TxManager
	Clock      Clock
}

// ListWorkspaces returns the whole active membership collection, by name then
// ID, restricted to candidates locked during this request. New memberships
// discovered on reread wait for the next request; revoked ones are absent.
type ListWorkspaces struct{ d ListWorkspacesDeps }

func NewListWorkspaces(d ListWorkspacesDeps) *ListWorkspaces { return &ListWorkspaces{d: d} }

func (u *ListWorkspaces) Execute(ctx context.Context) ([]domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	candidates, err := u.d.Workspaces.ListWorkspaces(ctx, actor.UserID)
	if err != nil {
		return nil, err
	}
	var out []domain.Workspace
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		locked, err := u.d.Workspaces.ShareImageWorkspaces(ctx, imageWorkspaceIDs(candidates))
		if err != nil {
			return err
		}
		allowed := make(map[uuid.UUID]bool, len(locked))
		for _, id := range locked {
			allowed[id] = true
		}
		current, err := u.d.Workspaces.ListWorkspaces(ctx, actor.UserID)
		if err != nil {
			return err
		}
		var visible []domain.Workspace
		for _, w := range current {
			if allowed[w.ID] {
				visible = append(visible, w)
			}
		}
		out, err = withWorkspaceImages(ctx, u.d.Images, visible)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
