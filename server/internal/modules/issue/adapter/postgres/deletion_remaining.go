package postgresadapter

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// ProjectDeletionRemaining skips completed tables and resets the cursor only
// when moving to a later table. It is called only at a budget boundary.
func (s *Store) ProjectDeletionRemaining(ctx context.Context, r app.ProjectRemainder) (bool, error) {
	steps := app.DeletionRowSteps()
	index := slices.Index(steps, r.Step)
	if index < 0 {
		return false, errors.New("invalid deletion row step")
	}
	after := r.After
	for _, step := range steps[index:] {
		pending, err := s.queries(ctx).ProjectDeletionStepRemaining(ctx, gen.ProjectDeletionStepRemainingParams{Step: string(step), ProjectID: r.ProjectID, After: after})
		if err != nil || pending {
			return pending, err
		}
		after = uuid.UUID{}
	}
	return false, nil
}
func (s *Store) WorkspaceDraftDeletionRemaining(ctx context.Context, b app.WorkspaceDraftBatch) (bool, error) {
	return s.queries(ctx).WorkspaceDraftDeletionRemaining(ctx, gen.WorkspaceDraftDeletionRemainingParams{WorkspaceID: b.WorkspaceID, After: b.After})
}
