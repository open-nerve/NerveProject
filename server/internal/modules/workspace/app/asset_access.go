package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AssetIntent uint8

const (
	AssetRead AssetIntent = iota + 1
	AssetPrepare
	AssetRemove
)

type AssetRequest struct {
	ID     uuid.UUID
	Intent AssetIntent
}
type AssetHost struct{ WorkspaceID uuid.UUID }

var ErrAssetScopeChanged = errors.New("workspace asset host scope changed")

type AssetAccess struct{ auth shared.Authorizer }

func NewAssetAccess(auth shared.Authorizer) *AssetAccess { return &AssetAccess{auth: auth} }
func (a *AssetAccess) Inspect(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, error) {
	if q.ID == uuid.Nil() || q.Intent < AssetRead || q.Intent > AssetRemove {
		return AssetHost{}, errors.New("invalid workspace asset request")
	}
	action := domain.ActionUpdate
	if q.Intent == AssetRead {
		action = domain.ActionRead
	}
	if _, err := decide(ctx, a.auth, actor, action, q.ID, domain.ErrNotFound); err != nil {
		return AssetHost{}, err
	}
	return AssetHost{WorkspaceID: q.ID}, nil
}
func (a *AssetAccess) Check(ctx context.Context, actor shared.Actor, q AssetRequest, before AssetHost) error {
	h, err := a.Inspect(ctx, actor, q)
	if err != nil {
		return err
	}
	if h != before {
		return ErrAssetScopeChanged
	}
	return nil
}
