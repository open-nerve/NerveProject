package domain

import (
	"fmt"
	"uuid"
)

// CycleListItem is a cycle as its project's list answers it (M6 P4 spec
// R9): with its status, the counts of the work items its reader sees
// (R1), their undeleted assignees, each once, in id order, and the counts
// of its frozen progress, nil while it is not frozen and for a reader who
// sees only his own work items (R5).
type CycleListItem struct {
	ReadCycle
	Live        Counts
	AssigneeIDs []uuid.UUID
	Snapshot    *FrozenCounts
}

// ModuleListItem is a module as its project's list answers it (M6 P4 spec
// R9): with the counts of the work items its reader sees (R1).
type ModuleListItem struct {
	Module
	Live Counts
}

// CountsOf counts, by cycle or module, the work items of rows by the group
// of their state among groups, the project's (M6 P4 spec R1, R9): a target
// without a row has none, its counts zero. A state not among groups, or one
// outside the five groups, is an error, as in ProgressOf.
func CountsOf(rows []StateCount, groups map[uuid.UUID]StateGroup) (map[uuid.UUID]Counts, error) {
	out := map[uuid.UUID]Counts{}
	for _, r := range rows {
		group, ok := groups[r.StateID]
		if !ok {
			return nil, fmt.Errorf("%d work items of %s are in state %s, not one of the project's", r.Issues, r.TargetID, r.StateID)
		}
		c := out[r.TargetID]
		if err := c.add(group, r.Issues); err != nil {
			return nil, fmt.Errorf("%d work items of %s: %w", r.Issues, r.TargetID, err)
		}
		out[r.TargetID] = c
	}
	return out, nil
}
