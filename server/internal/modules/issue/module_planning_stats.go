package issue

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// PlanningStats is what the module tells the planning module about the
// work items of a project's cycles and modules, which planning computes
// their progress and their list rows from (M6 P4 spec 2.1, R1, R2). Each
// reads in the transaction ctx carries: planning's read-only snapshot, or
// its write under the project's lock. A work item counts in a cycle or a
// module by an undeleted membership of it, when it is the project's,
// undeleted and not archived; viewer is an OwnOnly caller, who counts only
// the work items he created, none whose creator is gone, and nil counts
// every one. States are answered by id: their groups are the project
// module's. issue.Provide builds it over the pool; bootstrap converts it
// into planning's IssueStats.
type PlanningStats interface {
	// CycleItems reads the cycle's work items, in id order, each with its
	// state, its completion and its undeleted assignees and labels, each
	// once, in id order.
	CycleItems(ctx context.Context, projectID, cycleID uuid.UUID, viewer *uuid.UUID) ([]PlanningItem, error)
	// ModuleItems is CycleItems of a module's work items.
	ModuleItems(ctx context.Context, projectID, moduleID uuid.UUID, viewer *uuid.UUID) ([]PlanningItem, error)
	// CycleStateCounts counts, in one statement for the page of cycles,
	// each one's work items by state, by cycle and state in id order.
	CycleStateCounts(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]PlanningStateCount, error)
	// ModuleStateCounts is CycleStateCounts of a page of modules.
	ModuleStateCounts(ctx context.Context, projectID uuid.UUID, moduleIDs []uuid.UUID, viewer *uuid.UUID) ([]PlanningStateCount, error)
	// CycleAssignees reads, in one statement for the page of cycles, the
	// undeleted assignees of each one's work items, each once, by cycle and
	// assignee in id order.
	CycleAssignees(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]PlanningAssignee, error)
}

// PlanningItem, PlanningStateCount and PlanningAssignee are the values
// PlanningStats hands over: bootstrap converts them into planning's.
type (
	PlanningItem       = app.PlanningItem
	PlanningStateCount = app.PlanningStateCount
	PlanningAssignee   = app.PlanningAssignee
)
