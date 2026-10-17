package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
)

// StateInUse reports whether an undeleted work item, archived or not, is
// in the state (M4 design 3.6).
func (s *Store) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	inUse, err := s.queries(ctx).StateInUse(ctx, stateID)
	if err != nil {
		return false, fmt.Errorf("read whether state %s is in use: %w", stateID, err)
	}
	return inUse, nil
}

// DetachLabels soft-deletes the undeleted rows that put the labels on
// work items, at now by by (M4 design 3.6).
func (s *Store) DetachLabels(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DetachLabels(ctx, gen.DetachLabelsParams{At: now, DeletedBy: by, LabelIds: labelIDs}); err != nil {
		return fmt.Errorf("detach labels %v from their work items: %w", labelIDs, err)
	}
	return nil
}
