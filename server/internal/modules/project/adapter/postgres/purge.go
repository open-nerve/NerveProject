package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// PurgeProjectUserProperties commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeProjectUserProperties(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeProjectUserProperties(ctx, gen.PurgeProjectUserPropertiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge project_user_properties: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeProjectMembers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeProjectMembers(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeProjectMembers(ctx, gen.PurgeProjectMembersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge project_members: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeLabels commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeLabels(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	row, err := gen.New(s.pool).PurgeLabels(ctx, gen.PurgeLabelsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return app.PurgePage{After: b.After}, fmt.Errorf("purge labels: %w", err)
	}
	return app.PurgePage{Scanned: int(row.Scanned), Deleted: int(row.Deleted), After: row.AfterID}, nil
}

// PurgeStates scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeStates(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeStatesCandidates(ctx, gen.PurgeStatesCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeStates(ctx, gen.DeletePurgeStatesParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge states: %w", err)
	}
	return app.PurgePage(page), nil
}

// PurgeProjects scans candidates before deleting, retaining blocked progress.
func (s *Store) PurgeProjects(ctx context.Context, b app.PurgeBatch) (app.PurgePage, error) {
	q := gen.New(s.pool)
	page, err := postgres.PurgeProtected(ctx, protectedPurgeQueries{
		candidates: func(ctx context.Context, cutoff time.Time, after uuid.UUID, limit int) ([]uuid.UUID, error) {
			return q.PurgeProjectsCandidates(ctx, gen.PurgeProjectsCandidatesParams{Cutoff: &cutoff, AfterID: after, BatchLimit: int32(limit)})
		},
		remove: func(ctx context.Context, cutoff time.Time, ids []uuid.UUID) ([]uuid.UUID, error) {
			return q.DeletePurgeProjects(ctx, gen.DeletePurgeProjectsParams{Cutoff: &cutoff, Ids: ids})
		},
	}, s.protected, b.Cutoff, b.After, b.Limit)
	if err != nil {
		return app.PurgePage(page), fmt.Errorf("purge projects: %w", err)
	}
	return app.PurgePage(page), nil
}
