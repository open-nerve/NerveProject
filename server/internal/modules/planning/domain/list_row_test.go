package domain_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// A list row's counts are its work items by their state's group, as a progress counts them (M6 P4 spec R1, R9): the
// counts of a target's rows add up, two states of one group together, each row by its number; each target its own; a
// target without a row is not among them, its counts zero.
func TestCountsOf(t *testing.T) {
	sprint, release, empty := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	got, err := domain.CountsOf([]domain.StateCount{
		{TargetID: sprint, StateID: backlog, Issues: 2}, {TargetID: sprint, StateID: completed, Issues: 1},
		{TargetID: sprint, StateID: completed2, Issues: 3}, {TargetID: sprint, StateID: cancelled, Issues: 1},
		{TargetID: release, StateID: started, Issues: 4}, {TargetID: release, StateID: unstarted, Issues: 1},
	}, stateGroups())
	if err != nil {
		t.Fatal(err)
	}
	want := map[uuid.UUID]domain.Counts{
		sprint:  {Total: 7, Backlog: 2, Completed: 4, Cancelled: 1},
		release: {Total: 5, Unstarted: 1, Started: 4},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v\nwant %+v", got, want)
	}
	if c := got[empty]; c != (domain.Counts{}) {
		t.Fatalf("a target without work items counts %+v", c)
	}
	if got, err := domain.CountsOf(nil, stateGroups()); err != nil || len(got) != 0 {
		t.Fatalf("no rows: %+v, %v; want none", got, err)
	}
}

// A state outside the five groups takes no work item, in a list's counts as in a progress (M6 P4 spec R1): a row of
// the triage group, of an unknown group, or of a state not the project's is an error, never left out of the numbers.
func TestCountsOfRefusesAStateOutsideTheGroups(t *testing.T) {
	for _, tt := range []struct {
		name  string
		group domain.StateGroup
		state uuid.UUID
	}{
		{"triage", "triage", triage},
		{"an unknown group", "archived", triage},
		{"not the project's state", "", uuid.NewV7()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			groups := stateGroups()
			if tt.state == triage {
				groups[triage] = tt.group
			}
			target := uuid.NewV7()
			rows := []domain.StateCount{{TargetID: target, StateID: started, Issues: 1}, {TargetID: target, StateID: tt.state, Issues: 2}}
			if got, err := domain.CountsOf(rows, groups); err == nil {
				t.Fatalf("answered %+v", got)
			}
		})
	}
}
