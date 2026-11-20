package postgresadapter

import (
	"context"
	"fmt"
	"math"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
)

// PurgeIssueViews commits one bounded page of the views' purge (purge.sql)
// on the pool, so it commits independently of any caller's transaction.
func (s *Store) PurgeIssueViews(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeIssueViews(ctx, gen.PurgeIssueViewsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge issue views: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}
