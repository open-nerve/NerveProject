package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// CycleItems reads, in the transaction ctx carries, the work items of the
// cycle cycleID of the project projectID that planning computes its
// progress from (M6 P4 spec 2.1, R1, R2; planning_stats.sql): those with
// an undeleted membership of it, undeleted and not archived, of an OwnOnly
// viewer only those he created, viewer nil for every one; in id order,
// each with its undeleted assignees and labels, each once, in id order.
func (s *Store) CycleItems(ctx context.Context, projectID, cycleID uuid.UUID, viewer *uuid.UUID) ([]app.PlanningItem, error) {
	rows, err := s.queries(ctx).CycleItems(ctx, gen.CycleItemsParams{CycleID: cycleID, ProjectID: projectID, Viewer: viewer})
	if err != nil {
		return nil, fmt.Errorf("read the work items of cycle %s: %w", cycleID, err)
	}
	items := make([]app.PlanningItem, len(rows))
	for n, r := range rows {
		items[n] = planningItem(r)
	}
	return items, nil
}

// ModuleItems is CycleItems of the module moduleID's work items.
func (s *Store) ModuleItems(ctx context.Context, projectID, moduleID uuid.UUID, viewer *uuid.UUID) ([]app.PlanningItem, error) {
	rows, err := s.queries(ctx).ModuleItems(ctx, gen.ModuleItemsParams{ModuleID: moduleID, ProjectID: projectID, Viewer: viewer})
	if err != nil {
		return nil, fmt.Errorf("read the work items of module %s: %w", moduleID, err)
	}
	items := make([]app.PlanningItem, len(rows))
	for n, r := range rows {
		items[n] = planningItem(gen.CycleItemsRow(r))
	}
	return items, nil
}

func planningItem(r gen.CycleItemsRow) app.PlanningItem {
	return app.PlanningItem{ID: r.ID, StateID: r.StateID, CompletedAt: r.CompletedAt, AssigneeIDs: r.AssigneeIds, LabelIDs: r.LabelIds}
}

// CycleStateCounts counts, in one statement, the work items CycleItems
// reads of each of the cycles cycleIDs that is the project's, by state:
// by cycle and state in id order, none of a cycle without any (M6 P4 spec
// R2, R9).
func (s *Store) CycleStateCounts(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]app.PlanningStateCount, error) {
	rows, err := s.queries(ctx).CycleStateCounts(ctx, gen.CycleStateCountsParams{CycleIds: cycleIDs, ProjectID: projectID, Viewer: viewer})
	if err != nil {
		return nil, fmt.Errorf("count the work items of %d cycles: %w", len(cycleIDs), err)
	}
	counts := make([]app.PlanningStateCount, len(rows))
	for n, r := range rows {
		counts[n] = app.PlanningStateCount(r)
	}
	return counts, nil
}

// ModuleStateCounts is CycleStateCounts of the modules moduleIDs: a work
// item in several of them counts in each.
func (s *Store) ModuleStateCounts(ctx context.Context, projectID uuid.UUID, moduleIDs []uuid.UUID, viewer *uuid.UUID) ([]app.PlanningStateCount, error) {
	rows, err := s.queries(ctx).ModuleStateCounts(ctx, gen.ModuleStateCountsParams{ModuleIds: moduleIDs, ProjectID: projectID, Viewer: viewer})
	if err != nil {
		return nil, fmt.Errorf("count the work items of %d modules: %w", len(moduleIDs), err)
	}
	counts := make([]app.PlanningStateCount, len(rows))
	for n, r := range rows {
		counts[n] = app.PlanningStateCount(r)
	}
	return counts, nil
}

// CycleAssignees reads, in one statement, the undeleted assignees of the
// work items CycleItems reads of each of the cycles cycleIDs that is the
// project's, each once, by cycle and assignee in id order (M6 P4 spec R9).
func (s *Store) CycleAssignees(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]app.PlanningAssignee, error) {
	rows, err := s.queries(ctx).CycleAssignees(ctx, gen.CycleAssigneesParams{CycleIds: cycleIDs, ProjectID: projectID, Viewer: viewer})
	if err != nil {
		return nil, fmt.Errorf("read the assignees of %d cycles: %w", len(cycleIDs), err)
	}
	assignees := make([]app.PlanningAssignee, len(rows))
	for n, r := range rows {
		assignees[n] = app.PlanningAssignee(r)
	}
	return assignees, nil
}
