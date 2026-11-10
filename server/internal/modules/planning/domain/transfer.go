package domain

import (
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PendingStates are the states, by id in id order, among a project's
// states' groups, whose work items a transfer moves (M6 P4 spec 2.6, R7):
// those of the backlog, unstarted and started groups, which a progress
// counts as pending (Tally).
func PendingStates(groups map[uuid.UUID]StateGroup) []uuid.UUID {
	var ids []uuid.UUID
	for id, g := range groups {
		if g == GroupBacklog || g == GroupUnstarted || g == GroupStarted {
			ids = append(ids, id)
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids
}

// TransferTargetNotAllowed is the problem of a transfer's target that is
// not another undeleted cycle of the transferred cycle's project, archived
// or not: one that is not there or is deleted, another project's, or the
// transferred cycle itself (M6 P4 spec 2.3, R7). It is no 404: it tells
// nothing of a cycle the caller may not see.
func TransferTargetNotAllowed() shared.FieldError {
	return shared.FieldError{Field: "target_cycle_id", Code: shared.FieldNotAllowed, Message: "must be another cycle of the project"}
}
