package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// IssueStats reads, of the issue module, the facts of the work items of a
// project's cycles and modules that their progress and their list rows
// are computed from (M6 P4 spec 2.1, R1, R2), in the transaction ctx
// carries: a read's read-only snapshot, or a freeze's or a transfer's write
// under the project's lock. A work item counts in a cycle or a module by an
// undeleted membership of it, when it is the project's, undeleted and not
// archived, a sub-item too. viewer is an OwnOnly caller, who counts only
// the work items he created, none whose creator is gone (the list's
// predicate); nil counts every one. States come by id: the project's
// StateGroups give their groups. A cycle or a module that is another
// project's has none. issue.Provide's PlanningStats, converted (bootstrap).
type IssueStats interface {
	// CycleItems reads the cycle's work items, in id order, each with its
	// undeleted assignees and labels, each once, in id order.
	CycleItems(ctx context.Context, projectID, cycleID uuid.UUID, viewer *uuid.UUID) ([]domain.WorkItem, error)
	// ModuleItems is CycleItems of a module's work items.
	ModuleItems(ctx context.Context, projectID, moduleID uuid.UUID, viewer *uuid.UUID) ([]domain.WorkItem, error)
	// CycleCounts counts, in one read for the page of cycles, each one's
	// work items by state, by cycle and state in id order; a cycle without
	// any has no row.
	CycleCounts(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]domain.StateCount, error)
	// ModuleCounts is CycleCounts of a page of modules: a work item in
	// several counts in each.
	ModuleCounts(ctx context.Context, projectID uuid.UUID, moduleIDs []uuid.UUID, viewer *uuid.UUID) ([]domain.StateCount, error)
	// CycleAssignees reads, in one read for the page of cycles, the
	// undeleted assignees of each one's work items, each once, by cycle
	// and assignee in id order.
	CycleAssignees(ctx context.Context, projectID uuid.UUID, cycleIDs []uuid.UUID, viewer *uuid.UUID) ([]domain.CycleAssignee, error)
}
