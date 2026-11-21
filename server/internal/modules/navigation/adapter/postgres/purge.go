package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
)

// purgePage runs one bounded page of table's purge statement (purge.sql) on
// the pool, so it commits independently of any caller's transaction.
func (s *Store) purgePage(b app.PurgeBatch, table string,
	page func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (scanned, deleted int32, last uuid.UUID, err error)) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	scanned, deleted, last, err := page(gen.New(s.pool), &b.Cutoff, b.After, int32(b.Limit))
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge %s: %w", table, err)
	}
	return app.PurgePage{Scanned: int(scanned), Deleted: int(deleted), After: last}, nil
}

// PurgeIssueViews commits one bounded page of the views' purge.
func (s *Store) PurgeIssueViews(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "issue views", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeIssueViews(ctx, gen.PurgeIssueViewsParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}

// PurgeUserFavorites commits one bounded page of the favorites' purge.
func (s *Store) PurgeUserFavorites(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "user favorites", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeUserFavorites(ctx, gen.PurgeUserFavoritesParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}
