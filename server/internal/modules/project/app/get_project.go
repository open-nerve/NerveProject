package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type GetProjectDeps struct {
	Projects ProjectReader
	Auth     shared.Authorizer
	Locks    ImageReadLocks
	Gate     ImageGate
	Images   ImageReader
	Tx       shared.TxManager
}

// GetProject rechecks the visible candidate under account/workspace/project
// locks before signing. The gate is acquired without a connection or row lock.
type GetProject struct{ d GetProjectDeps }

func NewGetProject(d GetProjectDeps) *GetProject { return &GetProject{d: d} }

func (u *GetProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	p, found, err := u.d.Projects.GetProject(ctx, id, actor.UserID)
	switch {
	case err != nil:
		return domain.Project{}, err
	case !found:
		return domain.Project{}, domain.ErrNotFound
	}
	if _, err = decide(ctx, u.d.Auth, actor, domain.ActionRead, p.WorkspaceID, p.ID, domain.ErrNotFound); err != nil {
		return domain.Project{}, err
	}
	leave, err := u.d.Gate.Enter(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	defer leave()
	var out domain.Project
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, err := u.d.Locks.lock(ctx, actor, p.WorkspaceID, []uuid.UUID{id}, domain.ErrNotFound)
		if err != nil {
			return err
		}
		if locked[id] != p.WorkspaceID {
			return domain.ErrNotFound
		}
		current, found, err := u.d.Projects.GetProject(ctx, id, actor.UserID)
		if err != nil {
			return err
		}
		if !found || current.WorkspaceID != p.WorkspaceID {
			return domain.ErrNotFound
		}
		if _, err := decide(ctx, u.d.Auth, actor, domain.ActionRead, p.WorkspaceID, id, domain.ErrNotFound); err != nil {
			return err
		}
		images, err := withProjectImages(ctx, u.d.Images, []domain.Project{current})
		if err != nil {
			return err
		}
		out = images[0]
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return out, nil
}
