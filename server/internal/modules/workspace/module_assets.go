package workspace

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AssetRequest = app.AssetRequest
type AssetHost = app.AssetHost
type AssetParent = app.AssetParent
type AssetIntent = app.AssetIntent

const (
	AssetRead    = app.AssetRead
	AssetPrepare = app.AssetPrepare
	AssetRemove  = app.AssetRemove
)

// AssetAccess owns authorization. Callers hold parent locks before Check.
type AssetAccess interface {
	Inspect(context.Context, shared.Actor, AssetRequest) (AssetHost, error)
	Check(context.Context, shared.Actor, AssetRequest, AssetHost) error
}

func (m *Module) AssetAccess() AssetAccess { return m.assets }

// AssetParents uses the caller's transaction; fact reads take no row locks.
type AssetParents interface {
	ShareAssetWorkspaces(context.Context, []uuid.UUID) ([]uuid.UUID, error)
	AssetWorkspaceFacts(context.Context, []uuid.UUID) ([]AssetParent, error)
}

func AssetHostNotFound(err error) bool { return errors.Is(err, domain.ErrNotFound) }
func AssetHostChanged(err error) bool  { return errors.Is(err, app.ErrAssetScopeChanged) }
