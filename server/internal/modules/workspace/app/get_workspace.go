package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type GetWorkspaceRows interface {
	WorkspaceFinder
	ImageWorkspaceSharer
}

type GetWorkspaceDeps struct {
	Workspaces GetWorkspaceRows
	Auth       shared.Authorizer
	Caller     CallerLock
	Images     ImageReader
	Tx         shared.TxManager
	Clock      Clock
}

// GetWorkspace rechecks the visible candidate under account and workspace
// locks before issuing its logo capability.
type GetWorkspace struct{ d GetWorkspaceDeps }

func NewGetWorkspace(d GetWorkspaceDeps) *GetWorkspace { return &GetWorkspace{d: d} }

func (u *GetWorkspace) Execute(ctx context.Context, slug string) (domain.Workspace, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Workspace{}, err
	}
	candidate, err := u.read(ctx, actor, slug)
	if err != nil {
		return domain.Workspace{}, err
	}
	var out domain.Workspace
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		locked, err := u.d.Workspaces.ShareImageWorkspaces(ctx, []uuid.UUID{candidate.ID})
		if err != nil {
			return err
		}
		if !slices.Contains(locked, candidate.ID) {
			return domain.ErrNotFound
		}
		w, err := u.read(ctx, actor, slug)
		if err != nil {
			return err
		}
		if w.ID != candidate.ID {
			return domain.ErrNotFound
		}
		list, err := withWorkspaceImages(ctx, u.d.Images, []domain.Workspace{w})
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

func (u *GetWorkspace) read(ctx context.Context, actor shared.Actor, slug string) (domain.Workspace, error) {
	w, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
	if errors.Is(err, ErrNotFound) {
		return domain.Workspace{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Workspace{}, err
	}
	grant, err := decide(ctx, u.d.Auth, actor, domain.ActionRead, w.ID, domain.ErrNotFound)
	if err != nil {
		return domain.Workspace{}, err
	}
	w.Role = grant.WorkspaceRole
	return w, nil
}
