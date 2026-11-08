package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleTargets reads those of the cycles ids that are the project's and
// undeleted, archived or not, in id order (issue_targets.sql).
func (s *Store) CycleTargets(ctx context.Context, project uuid.UUID, ids []uuid.UUID) ([]app.IssueTarget, error) {
	rows, err := s.queries(ctx).CycleTargets(ctx, gen.CycleTargetsParams{ProjectID: project, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the cycles of project %s a work item names: %w", project, err)
	}
	out := make([]app.IssueTarget, len(rows))
	for i, r := range rows {
		out[i] = app.IssueTarget(r)
	}
	return out, nil
}

// ModuleTargets reads those of the modules ids that are the project's and
// undeleted, archived or not, in id order (issue_targets.sql).
func (s *Store) ModuleTargets(ctx context.Context, project uuid.UUID, ids []uuid.UUID) ([]app.IssueTarget, error) {
	rows, err := s.queries(ctx).ModuleTargets(ctx, gen.ModuleTargetsParams{ProjectID: project, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the modules of project %s a work item names: %w", project, err)
	}
	out := make([]app.IssueTarget, len(rows))
	for i, r := range rows {
		out[i] = app.IssueTarget(r)
	}
	return out, nil
}

// OpenCycleIDs reads the ids of the project's undeleted cycles, archived or
// not, that have not ended on today, in id order (issue_targets.sql).
func (s *Store) OpenCycleIDs(ctx context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).OpenCycleIDs(ctx, gen.OpenCycleIDsParams{ProjectID: project, Today: today.Time()})
	if err != nil {
		return nil, fmt.Errorf("read the open cycles of project %s on %s: %w", project, today, err)
	}
	return ids, nil
}

// OpenModuleIDs reads the ids of the project's undeleted modules, archived
// or not and in any status, whose target day has not passed on today, in id
// order (issue_targets.sql).
func (s *Store) OpenModuleIDs(ctx context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).OpenModuleIDs(ctx, gen.OpenModuleIDsParams{ProjectID: project, Today: today.Time()})
	if err != nil {
		return nil, fmt.Errorf("read the open modules of project %s on %s: %w", project, today, err)
	}
	return ids, nil
}
