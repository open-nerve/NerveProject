package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// DeleteProjectCycles soft-deletes a batch of a deleted project's cycles,
// archived ones too (app.ProjectRows, delete_rows.sql).
func (s *Store) DeleteProjectCycles(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectCycles(ctx, gen.DeleteProjectCyclesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's cycles: %w", err)
	}
	return ids, nil
}

// DeleteProjectModules soft-deletes a batch of a deleted project's modules,
// archived ones too.
func (s *Store) DeleteProjectModules(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectModules(ctx, gen.DeleteProjectModulesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's modules: %w", err)
	}
	return ids, nil
}

// DeleteProjectModuleMembers soft-deletes a batch of a deleted project's
// modules' members.
func (s *Store) DeleteProjectModuleMembers(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectModuleMembers(ctx, gen.DeleteProjectModuleMembersParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's module members: %w", err)
	}
	return ids, nil
}

// DeleteProjectModuleLinks soft-deletes a batch of a deleted project's
// modules' links.
func (s *Store) DeleteProjectModuleLinks(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectModuleLinks(ctx, gen.DeleteProjectModuleLinksParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's module links: %w", err)
	}
	return ids, nil
}

// DeleteProjectCycleUserProperties soft-deletes a batch of the accounts'
// display settings of a deleted project's cycles' pages.
func (s *Store) DeleteProjectCycleUserProperties(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectCycleUserProperties(ctx, gen.DeleteProjectCycleUserPropertiesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's cycle preferences: %w", err)
	}
	return ids, nil
}

// DeleteProjectModuleUserProperties soft-deletes a batch of the accounts'
// display settings of a deleted project's modules' pages.
func (s *Store) DeleteProjectModuleUserProperties(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectModuleUserProperties(ctx, gen.DeleteProjectModuleUserPropertiesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's module preferences: %w", err)
	}
	return ids, nil
}

// ProjectDeletionRemaining tells, without locking or writing, whether the
// project has undeleted rows in the remainder's step after its cursor, or in
// any later step; the steps before it are done. It is called only at a budget
// boundary.
func (s *Store) ProjectDeletionRemaining(ctx context.Context, r app.ProjectRemainder) (bool, error) {
	steps := app.DeletionRowSteps()
	index := slices.Index(steps, r.Step)
	if index < 0 {
		return false, errors.New("invalid planning deletion step")
	}
	after := r.After
	for _, step := range steps[index:] {
		pending, err := s.queries(ctx).ProjectDeletionStepRemaining(ctx, gen.ProjectDeletionStepRemainingParams{Step: string(step), ProjectID: r.ProjectID, After: after})
		if err != nil {
			return false, fmt.Errorf("read the project's remaining %s: %w", step, err)
		}
		if pending {
			return true, nil
		}
		after = uuid.UUID{}
	}
	return false, nil
}
