package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type AssetReader interface {
	FindAsset(context.Context, uuid.UUID) (domain.Asset, error)
}
type AssetCreator interface {
	CreateAsset(context.Context, domain.Asset) (domain.Asset, error)
}

// SaveAsset compares StorageRevision and never changes the applicant or scope.
// Callers hold the asset lock; the revision also rejects stale external results.
type MetadataWriter interface {
	SaveAsset(context.Context, domain.Asset) (domain.Asset, error)
}
type RecoveryReader interface {
	ListRecovery(context.Context, uuid.UUID, int) ([]domain.Asset, error)
}
type ScopeReader interface {
	ListScope(context.Context, domain.ScopeKind, uuid.UUID, uuid.UUID, int) ([]domain.Asset, error)
}
type ScopeDeletions interface {
	DeleteScopeRows(context.Context, domain.ScopeKind, uuid.UUID, DeletionStamp) error
}
type PurgePage struct {
	Scanned, Deleted int
	After            uuid.UUID
}
type PurgeRows interface {
	PurgeAssets(context.Context, time.Time, uuid.UUID, int) (PurgePage, error)
}
