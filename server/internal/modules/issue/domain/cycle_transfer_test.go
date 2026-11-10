package domain

import (
	"reflect"
	"testing"
	"uuid"
)

// A work item a transfer moves out of a cycle that has ended into another (M6 P4 spec 2.6, R8) writes the row of its
// moving between them: updated, from the name of the cycle it leaves to the name of the one it joins, the first's id
// the old identifier and the second's the new, Plane's sentence cut before the names: the row a PATCH that moves it
// from the one to the other writes.
func TestMovingBetweenCycles(t *testing.T) {
	sprint, next := Named{ID: uuid.NewV7(), Name: "Sprint 1"}, Named{ID: uuid.NewV7(), Name: "Sprint 2"}
	field, one, two := "cycles", "Sprint 1", "Sprint 2"
	want := Activity{Field: &field, Verb: VerbUpdated, OldValue: &one, NewValue: &two, OldIdentifier: &sprint.ID, NewIdentifier: &next.ID,
		Comment: "updated cycle from "}
	patch := UpdateActivity(Update{Cycle: &Changed[*Named]{Old: &sprint, New: &next}})
	if got := CycleMoved(sprint, next); !reflect.DeepEqual(got, want) || !reflect.DeepEqual([]Activity{got}, patch) {
		t.Errorf("CycleMoved() = %+v; want %+v, as a PATCH's %+v", got, want, patch)
	}
}
