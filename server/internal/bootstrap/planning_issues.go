package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
)

// planningIssues is issue's PlanningStats as planning's IssueStats (M6 P4
// spec R2): the same reads, of the project, the cycles or modules and the
// viewer asked about, each row converted, in their order.
type planningIssues struct {
	stats issue.PlanningStats
}

func (p planningIssues) CycleItems(ctx context.Context, projectID, cycleID uuid.UUID, viewer *uuid.UUID) ([]planning.WorkItem, error) {
	items, err := p.stats.CycleItems(ctx, projectID, cycleID, viewer)
	if err != nil {
		return nil, err
	}
	return planningWorkItems(items), nil
}

func (p planningIssues) ModuleItems(ctx context.Context, projectID, moduleID uuid.UUID, viewer *uuid.UUID) ([]planning.WorkItem, error) {
	items, err := p.stats.ModuleItems(ctx, projectID, moduleID, viewer)
	if err != nil {
		return nil, err
	}
	return planningWorkItems(items), nil
}

// planningWorkItems is issue's work items as planning's, in their order.
func planningWorkItems(items []issue.PlanningItem) []planning.WorkItem {
	out := make([]planning.WorkItem, len(items))
	for i, item := range items {
		out[i] = planning.WorkItem(item)
	}
	return out
}

func (p planningIssues) CycleCounts(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]planning.StateCount, error) {
	counts, err := p.stats.CycleStateCounts(ctx, projectID, cycleIDs, viewer)
	if err != nil {
		return nil, err
	}
	return planningStateCounts(counts), nil
}

func (p planningIssues) ModuleCounts(ctx context.Context, projectID uuid.UUID, moduleIDs []uuid.UUID, viewer *uuid.UUID) ([]planning.StateCount, error) {
	counts, err := p.stats.ModuleStateCounts(ctx, projectID, moduleIDs, viewer)
	if err != nil {
		return nil, err
	}
	return planningStateCounts(counts), nil
}

// planningStateCounts is issue's counts by state as planning's, in their
// order.
func planningStateCounts(counts []issue.PlanningStateCount) []planning.StateCount {
	out := make([]planning.StateCount, len(counts))
	for i, c := range counts {
		out[i] = planning.StateCount(c)
	}
	return out
}

func (p planningIssues) CycleAssignees(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]planning.CycleAssignee, error) {
	assignees, err := p.stats.CycleAssignees(ctx, projectID, cycleIDs, viewer)
	if err != nil {
		return nil, err
	}
	out := make([]planning.CycleAssignee, len(assignees))
	for i, a := range assignees {
		out[i] = planning.CycleAssignee(a)
	}
	return out, nil
}

// planningTransfer is issue's PlanningTransfer as planning's IssueTransfer
// (M6 P4 spec R2): the same move, in the context it comes in, which carries
// planning's transaction, every field converted; issue's count and error as
// they come.
type planningTransfer struct {
	transfer issue.PlanningTransfer
}

func (p planningTransfer) TransferCycleIssues(ctx context.Context, t planning.Transfer) (int, error) {
	return p.transfer.TransferCycleIssues(ctx, issue.CycleTransfer{ProjectID: t.ProjectID, WorkspaceID: t.WorkspaceID, From: issue.Named(t.From),
		To: issue.Named(t.To), StateIDs: t.StateIDs, By: t.By, At: t.At})
}
