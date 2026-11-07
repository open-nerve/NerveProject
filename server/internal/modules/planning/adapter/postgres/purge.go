package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
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

// PurgeCycles commits one bounded page of the cycles' purge.
func (s *Store) PurgeCycles(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "cycles", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeCycles(ctx, gen.PurgeCyclesParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}

// PurgeModules commits one bounded page of the modules' purge.
func (s *Store) PurgeModules(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "modules", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeModules(ctx, gen.PurgeModulesParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}

// PurgeModuleMembers commits one bounded page of the module members' purge.
func (s *Store) PurgeModuleMembers(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "module_members", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeModuleMembers(ctx, gen.PurgeModuleMembersParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}

// PurgeModuleLinks commits one bounded page of the module links' purge.
func (s *Store) PurgeModuleLinks(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	return s.purgePage(b, "module_links", func(q *gen.Queries, cutoff *time.Time, after uuid.UUID, limit int32) (int32, int32, uuid.UUID, error) {
		row, err := q.PurgeModuleLinks(ctx, gen.PurgeModuleLinksParams{Cutoff: cutoff, AfterID: after, BatchLimit: limit})
		return row.Scanned, row.Deleted, row.AfterID, err
	})
}
