package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AssetIntent uint8

const (
	AssetRead AssetIntent = iota + 1
	AssetPrepare
	AssetRemove
)

type AssetRequest struct {
	ID         uuid.UUID
	NewProject bool
	Intent     AssetIntent
}
type AssetHost struct{ WorkspaceID, ProjectID uuid.UUID }

var ErrAssetScopeChanged = errors.New("project asset host scope changed")

type AssetAccess struct {
	projects ProjectReader
	auth     shared.Authorizer
}

func NewAssetAccess(projects ProjectReader, auth shared.Authorizer) *AssetAccess {
	return &AssetAccess{projects: projects, auth: auth}
}
func (a *AssetAccess) Inspect(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, error) {
	h, _, err := a.read(ctx, actor, q)
	return h, err
}
func (a *AssetAccess) Check(ctx context.Context, actor shared.Actor, q AssetRequest, before AssetHost) error {
	h, archived, err := a.read(ctx, actor, q)
	if err != nil {
		return err
	}
	if h != before {
		return ErrAssetScopeChanged
	}
	if q.Intent != AssetRead && archived {
		return domain.ErrArchived
	}
	return nil
}

func (a *AssetAccess) read(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, bool, error) {
	if q.ID == uuid.Nil() || q.Intent < AssetRead || q.Intent > AssetRemove {
		return AssetHost{}, false, errors.New("invalid project asset request")
	}
	if q.NewProject {
		if _, err := decide(ctx, a.auth, actor, domain.ActionCreate, q.ID, uuid.Nil(), domain.ErrWorkspaceNotFound); err != nil {
			return AssetHost{}, false, err
		}
		return AssetHost{WorkspaceID: q.ID}, false, nil
	}
	p, found, err := a.projects.GetProject(ctx, q.ID, actor.UserID)
	if err != nil {
		return AssetHost{}, false, err
	}
	if !found {
		return AssetHost{}, false, domain.ErrNotFound
	}
	action := domain.ActionUpdate
	if q.Intent == AssetRead {
		action = domain.ActionRead
	}
	if _, err := decide(ctx, a.auth, actor, action, p.WorkspaceID, p.ID, domain.ErrNotFound); err != nil {
		return AssetHost{}, false, err
	}
	return AssetHost{WorkspaceID: p.WorkspaceID, ProjectID: p.ID}, p.ArchivedAt != nil, nil
}
