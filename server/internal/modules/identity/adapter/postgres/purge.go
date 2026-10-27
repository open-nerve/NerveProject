package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// PurgeAPITokens commits one batch on the pool, independently of any request
// transaction in ctx. A failed statement returns no committed IDs.
func (s *Store) PurgeAPITokens(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeAPITokens(ctx, gen.PurgeAPITokensParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge API tokens: %w", err)
	}
	return ids, nil
}
