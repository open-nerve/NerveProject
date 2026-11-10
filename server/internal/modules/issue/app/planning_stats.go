package app

import (
	"time"
	"uuid"
)

// PlanningItem is a work item of a cycle or of a module as the module tells
// planning about it, which computes the progress from it (M6 P4 spec R2):
// its id, its state by id, the moment it was put in a completed state, if
// any, and the ids of its undeleted assignees and labels, each once, in id
// order. The state's group is the project module's, which planning reads;
// the issue module's statements do not read the states.
type PlanningItem struct {
	ID, StateID           uuid.UUID
	CompletedAt           *time.Time
	AssigneeIDs, LabelIDs []uuid.UUID
}

// PlanningStateCount is how many of a cycle's or a module's work items,
// TargetID's, are in the state StateID (M6 P4 spec R2): a list row's
// counts, which planning groups by the states' groups.
type PlanningStateCount struct {
	TargetID, StateID uuid.UUID
	Issues            int64
}

// PlanningAssignee is an account assigned to a work item of a cycle (M6 P4
// spec R9): a cycle list row's assignee ids.
type PlanningAssignee struct {
	CycleID, AssigneeID uuid.UUID
}
