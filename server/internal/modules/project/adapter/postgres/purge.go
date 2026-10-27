package postgresadapter

import (
	"context"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// PurgeProjectUserProperties commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeProjectUserProperties(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeProjectUserProperties(ctx, gen.PurgeProjectUserPropertiesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge project_user_properties: %w", err)
	}
	return ids, nil
}

// PurgeProjectMembers commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeProjectMembers(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeProjectMembers(ctx, gen.PurgeProjectMembersParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge project_members: %w", err)
	}
	return ids, nil
}

// PurgeLabels commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeLabels(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeLabels(ctx, gen.PurgeLabelsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge labels: %w", err)
	}
	return ids, nil
}

// PurgeStates commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeStates(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeStates(ctx, gen.PurgeStatesParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge states: %w", err)
	}
	return ids, nil
}

// PurgeProjects commits one bounded batch on the pool, independently of
// any request transaction carried in ctx.
func (s *Store) PurgeProjects(ctx context.Context, b app.PurgeBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("purge batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := gen.New(s.pool).PurgeProjects(ctx, gen.PurgeProjectsParams{Cutoff: &b.Cutoff, AfterID: b.After, BatchLimit: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("purge projects: %w", err)
	}
	return ids, nil
}
