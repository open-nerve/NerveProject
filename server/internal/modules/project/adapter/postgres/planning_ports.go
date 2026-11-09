package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// PlanningFacts reads the undeleted project id's facts for the planning
// module without a lock (planning_ports.sql); found is false when there is
// none.
func (s *Store) PlanningFacts(ctx context.Context, id uuid.UUID) (app.PlanningFacts, bool, error) {
	r, err := s.queries(ctx).PlanningProjectFacts(ctx, id)
	return planningFacts(gen.LockPlanningProjectRow(r), err, "read the planning facts of project", id)
}

// LockPlanning locks the undeleted project id FOR NO KEY UPDATE, its
// workspace not locked, and reads its facts; found is false when there is
// none, also when it was deleted while the lock waited.
func (s *Store) LockPlanning(ctx context.Context, id uuid.UUID) (app.PlanningFacts, bool, error) {
	r, err := s.queries(ctx).LockPlanningProject(ctx, id)
	return planningFacts(r, err, "lock project for its cycles and modules", id)
}

// SharePlanning locks the undeleted project id FOR SHARE, its workspace not
// locked, and reads its facts; found is false when there is none, also when
// it was deleted while the lock waited.
func (s *Store) SharePlanning(ctx context.Context, id uuid.UUID) (app.PlanningFacts, bool, error) {
	r, err := s.queries(ctx).SharePlanningProject(ctx, id)
	return planningFacts(gen.LockPlanningProjectRow(r), err, "share project for a page's display settings", id)
}

func planningFacts(r gen.LockPlanningProjectRow, err error, what string, id uuid.UUID) (app.PlanningFacts, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.PlanningFacts{}, false, nil
	case err != nil:
		return app.PlanningFacts{}, false, fmt.Errorf("%s %s: %w", what, id, err)
	}
	return app.PlanningFacts{WorkspaceID: r.WorkspaceID, Archived: r.Archived, Timezone: r.Timezone, CycleView: r.CycleView, ModuleView: r.ModuleView}, true, nil
}
