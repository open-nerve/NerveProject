package postgresadapter

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
)

// DeleteCyclePreferenceRows soft-deletes the deleted cycle's page's undeleted
// display settings, each account's row, locked in id order, deleted and last
// changed by by at at, in one statement (preferences.sql).
func (s *Store) DeleteCyclePreferenceRows(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteCyclePreferenceRows(ctx, gen.DeleteCyclePreferenceRowsParams{CycleID: cycle, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the display settings of cycle %s: %w", cycle, err)
	}
	return nil
}

// DeleteModulePreferenceRows is DeleteCyclePreferenceRows of the deleted
// module's page.
func (s *Store) DeleteModulePreferenceRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	err := s.queries(ctx).DeleteModulePreferenceRows(ctx, gen.DeleteModulePreferenceRowsParams{ModuleID: module, At: at, By: by})
	if err != nil {
		return fmt.Errorf("delete the display settings of module %s: %w", module, err)
	}
	return nil
}
