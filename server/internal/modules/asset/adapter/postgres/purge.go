package postgresadapter

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) PurgeAssetCandidates(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("invalid asset purge batch")
	}
	return gen.New(s.pool).PurgeAssetCandidates(ctx, gen.PurgeAssetCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
}
func (s *Store) DeletePurgeAssets(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("asset purge requires a transaction")
	}
	if len(ids) > 1000 {
		return nil, errors.New("invalid asset purge batch")
	}
	return s.queries(ctx).DeletePurgeAssets(ctx, gen.DeletePurgeAssetsParams{Cutoff: &cutoff, AssetIds: ids})
}
