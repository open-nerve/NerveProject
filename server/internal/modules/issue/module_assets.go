package issue

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
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

type AssetHostKind = app.AssetHostKind

const (
	AssetIssue      = app.AssetIssue
	AssetComment    = app.AssetComment
	AssetNewComment = app.AssetNewComment
	AssetDraft      = app.AssetDraft
)

// AssetAccess owns authorization. Callers hold parent locks before Check.
type AssetAccess interface {
	Inspect(context.Context, shared.Actor, AssetRequest) (AssetHost, error)
	Check(context.Context, shared.Actor, AssetRequest, AssetHost) error
}

func (m *Module) AssetAccess() AssetAccess { return m.assets }

// AssetParents uses the caller's transaction; fact reads take no row locks.
type AssetParents interface {
	LockAssetDrafts(context.Context, []uuid.UUID) ([]uuid.UUID, error)
	AssetIssueFacts(context.Context, []uuid.UUID) ([]AssetParent, error)
	AssetCommentFacts(context.Context, []uuid.UUID) ([]AssetParent, error)
	AssetDraftFacts(context.Context, []uuid.UUID) ([]AssetParent, error)
}

func AssetHostNotFound(err error) bool {
	return errors.Is(err, domain.ErrIssueNotFound) || errors.Is(err, domain.ErrCommentNotFound) || errors.Is(err, domain.ErrDraftNotFound) || errors.Is(err, domain.ErrWorkspaceNotFound)
}
func AssetHostChanged(err error) bool { return errors.Is(err, app.ErrAssetScopeChanged) }
