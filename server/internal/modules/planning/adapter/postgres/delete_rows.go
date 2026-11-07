package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// DeleteProjectCycles soft-deletes a batch of a deleted project's cycles,
// archived ones too (app.CycleRows, delete_rows.sql).
func (s *Store) DeleteProjectCycles(ctx context.Context, b app.CycleBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectCycles(ctx, gen.DeleteProjectCyclesParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's cycles: %w", err)
	}
	return ids, nil
}

// ProjectCyclesRemaining tells whether the project has undeleted cycles after
// the cursor, without locking or writing.
func (s *Store) ProjectCyclesRemaining(ctx context.Context, project, after uuid.UUID) (bool, error) {
	pending, err := s.queries(ctx).ProjectCyclesRemaining(ctx, gen.ProjectCyclesRemainingParams{ProjectID: project, After: after})
	if err != nil {
		return false, fmt.Errorf("read the project's remaining cycles: %w", err)
	}
	return pending, nil
}
