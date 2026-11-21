package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// NavigationCycles reads, in one statement and without a lock, the
// undeleted cycles among ids, archived or not, of any workspace, by cycle
// (navigation_targets.sql); a deleted one and an id of none are not among
// them. It reads in the transaction ctx carries or on the pool.
func (s *Store) NavigationCycles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.NavigationTarget, error) {
	rows, err := s.queries(ctx).NavigationCycles(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d cycles for favorites: %w", len(ids), err)
	}
	return navigationTargets(rows, func(r gen.NavigationCyclesRow) gen.NavigationModulesRow { return gen.NavigationModulesRow(r) }), nil
}

// NavigationModules reads the undeleted modules among ids as
// NavigationCycles reads the cycles.
func (s *Store) NavigationModules(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]app.NavigationTarget, error) {
	rows, err := s.queries(ctx).NavigationModules(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read %d modules for favorites: %w", len(ids), err)
	}
	return navigationTargets(rows, func(r gen.NavigationModulesRow) gen.NavigationModulesRow { return r }), nil
}

// navigationTargets are the cycles or the modules rows hold, by id.
func navigationTargets[R any](rows []R, row func(R) gen.NavigationModulesRow) map[uuid.UUID]app.NavigationTarget {
	targets := make(map[uuid.UUID]app.NavigationTarget, len(rows))
	for _, r := range rows {
		t := row(r)
		targets[t.ID] = app.NavigationTarget{WorkspaceID: t.WorkspaceID, ProjectID: t.ProjectID, Name: t.Name, Archived: t.Archived}
	}
	return targets
}
