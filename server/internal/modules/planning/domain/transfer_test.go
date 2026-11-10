package domain_test

import (
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// A transfer moves the work items of the backlog, unstarted and started groups (M6 P4 spec 2.6, R7): their states, in
// id order; not a completed, cancelled or triage one, nor one of a group outside the five.
func TestPendingStates(t *testing.T) {
	groups := stateGroups()
	groups[triage] = "triage"
	groups[uuid.MustParse("5a000000-0000-7000-8000-000000000000")] = "archived"
	if got, want := domain.PendingStates(groups), []uuid.UUID{backlog, unstarted, started}; !slices.Equal(got, want) {
		t.Errorf("PendingStates = %v, want %v", got, want)
	}
	if got := domain.PendingStates(nil); len(got) != 0 {
		t.Errorf("PendingStates of no state = %v, want none", got)
	}
}

// A transfer's target that is not another cycle of the project is refused at target_cycle_id (M6 P4 spec 2.3, R7).
func TestTransferTargetNotAllowed(t *testing.T) {
	if got := domain.TransferTargetNotAllowed(); got.Field != "target_cycle_id" || got.Code != "not_allowed" || got.Message == "" {
		t.Errorf("TransferTargetNotAllowed = %+v, want target_cycle_id not_allowed with a message", got)
	}
}
