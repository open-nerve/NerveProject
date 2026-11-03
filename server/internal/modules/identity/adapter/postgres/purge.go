package postgresadapter

import (
	"context"
	"fmt"
	"math"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// PurgeAPITokens commits one batch on the pool, independently of any request
// transaction in ctx. A failed statement returns no committed IDs.
func (s *Store) PurgeAPITokens(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeAPITokens(ctx, gen.PurgeAPITokensParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge API tokens: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}
