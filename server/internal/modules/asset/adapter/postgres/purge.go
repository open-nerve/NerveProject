package postgresadapter

import (
	"context"
	"errors"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
)

func (s *Store) PurgeAssets(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) (app.PurgePage, error) {
	page := app.PurgePage{After: after}
	if limit < 1 || limit > math.MaxInt32 {
		return page, errors.New("invalid asset purge batch")
	}
	row, err := gen.New(s.pool).PurgeAssets(ctx, gen.PurgeAssetsParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
	if err != nil {
		return page, err
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}
