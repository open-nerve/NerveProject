package domain

import (
	"slices"
	"testing"
)

// The six default states are Plane's, in its order: one per group, Backlog
// the only default, Triage the only triage state (M3 design 3.17).
func TestDefaultStates(t *testing.T) {
	want := []NewState{
		{"Backlog", "#60646C", 15000, "backlog", true},
		{"Todo", "#60646C", 25000, "unstarted", false},
		{"In Progress", "#F59E0B", 35000, "started", false},
		{"Done", "#46A758", 45000, "completed", false},
		{"Cancelled", "#9AA4BC", 55000, "cancelled", false},
		{"Triage", "#4E5355", 65000, "triage", false},
	}
	if got := DefaultStates(); !slices.Equal(got, want) {
		t.Errorf("DefaultStates() =\n%+v\nwant\n%+v", got, want)
	}
}
