package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ListProjectsDeps struct {
	Workspaces WorkspaceDirectory
	Projects   ProjectLister
	Auth       shared.Authorizer
	Locks      ImageReadLocks
	Gate       ImageGate
	Images     ImageReader
	Tx         shared.TxManager
}

// ListProjects keeps the original visibility and display order. Only candidates
// locked by this request can be returned; newly visible rows wait for a later read.
type ListProjects struct{ d ListProjectsDeps }

func NewListProjects(d ListProjectsDeps) *ListProjects { return &ListProjects{d: d} }

func (u *ListProjects) Execute(ctx context.Context, slug string, archived bool) ([]domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	ws, grant, err := findWorkspaceAndDecide(ctx, u.d.Workspaces, u.d.Auth, actor, slug, domain.ActionList)
	if err != nil {
		return nil, err
	}
	candidates, err := u.d.Projects.ListProjects(ctx, ws.ID, actor.UserID, domain.VisibilityOf(grant.WorkspaceRole), archived)
	if err != nil {
		return nil, err
	}
	ids := imageProjectIDs(candidates)
	leave, err := u.d.Gate.Enter(ctx, ids...)
	if err != nil {
		return nil, err
	}
	defer leave()
	var out []domain.Project
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, err := u.d.Locks.lock(ctx, actor, ws.ID, ids, domain.ErrWorkspaceNotFound)
		if err != nil {
			return err
		}
		currentWS, grant, err := findWorkspaceAndDecide(ctx, u.d.Workspaces, u.d.Auth, actor, slug, domain.ActionList)
		if err != nil {
			return err
		}
		if currentWS.ID != ws.ID {
			return domain.ErrWorkspaceNotFound
		}
		current, err := u.d.Projects.ListProjects(ctx, ws.ID, actor.UserID, domain.VisibilityOf(grant.WorkspaceRole), archived)
		if err != nil {
			return err
		}
		var visible []domain.Project
		for _, p := range current {
			if p.WorkspaceID == ws.ID && locked[p.ID] == ws.ID {
				visible = append(visible, p)
			}
		}
		out, err = withProjectImages(ctx, u.d.Images, visible)
		return err
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
