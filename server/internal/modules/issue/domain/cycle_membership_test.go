package domain

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A patch's cycle (M6 P3a spec 2.3): not given, or given as the work item
// has it, changes nothing; another cycle given moves the work item from its
// own, none for none, into it; null given takes it out of its own, and of
// none changes nothing. Changes.None knows the change.
func TestChangesOfTheCycle(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	in, out := Issue{CycleID: &a}, Issue{}
	for _, tt := range []struct {
		name string
		i    Issue
		p    IssuePatch
		want *CycleChange
	}{
		{"not given", in, IssuePatch{}, nil},
		{"the same", in, IssuePatch{SetCycleID: true, CycleID: ptr(a)}, nil},
		{"moved", in, IssuePatch{SetCycleID: true, CycleID: &b}, &CycleChange{From: &a, To: &b}},
		{"left", in, IssuePatch{SetCycleID: true}, &CycleChange{From: &a}},
		{"joined", out, IssuePatch{SetCycleID: true, CycleID: &b}, &CycleChange{To: &b}},
		{"none left", out, IssuePatch{SetCycleID: true}, nil},
	} {
		c := ChangesOf(tt.i, tt.p)
		if !reflect.DeepEqual(c.Cycle, tt.want) || c.None() != (tt.want == nil) {
			t.Errorf("%s: ChangesOf().Cycle = %+v, None() %t; want %+v, %t", tt.name, c.Cycle, c.None(), tt.want, tt.want == nil)
		}
	}
}

// The rows of a work item's cycle (M6 P3a spec 2.4, Plane's field and
// verbs): joining one is created, from "" to its name, its id the new
// identifier; moving to another is updated, from the old name to the new,
// both ids; leaving one is deleted, from its name to "", its id the old
// identifier. Plane's sentences go with them, without the names. An update
// writes the row after every other (UpdateActivity); a creation writes the
// cycle it joins after its own rows (CreationJoins), nothing for none.
func TestCycleActivity(t *testing.T) {
	sprint, next := Named{ID: uuid.NewV7(), Name: "Sprint 1"}, Named{ID: uuid.NewV7(), Name: "Sprint 2"}
	field, empty, one, two := "cycles", "", "Sprint 1", "Sprint 2"
	joined := Activity{Field: &field, Verb: VerbCreated, OldValue: &empty, NewValue: &one, NewIdentifier: &sprint.ID, Comment: "added cycle "}
	moved := Activity{Field: &field, Verb: VerbUpdated, OldValue: &one, NewValue: &two, OldIdentifier: &sprint.ID, NewIdentifier: &next.ID,
		Comment: "updated cycle from "}
	left := Activity{Field: &field, Verb: VerbDeleted, OldValue: &two, NewValue: &empty, OldIdentifier: &next.ID,
		Comment: "removed this work item from cycle "}
	for _, tt := range []struct {
		name   string
		change Changed[*Named]
		want   Activity
	}{
		{"joined", Changed[*Named]{New: &sprint}, joined},
		{"moved", Changed[*Named]{Old: &sprint, New: &next}, moved},
		{"left", Changed[*Named]{Old: &next}, left},
	} {
		if got := UpdateActivity(Update{Cycle: &tt.change}); !reflect.DeepEqual(got, []Activity{tt.want}) {
			t.Errorf("%s: UpdateActivity() = %+v, want %+v", tt.name, got, []Activity{tt.want})
		}
	}
	rows := UpdateActivity(Update{Name: &Changed[string]{Old: "Fix", New: "Mend"}, AssigneesRemoved: []Assignee{{ID: uuid.NewV7(), DisplayName: "bob"}},
		Cycle: &Changed[*Named]{Old: &sprint, New: &next}})
	if len(rows) != 3 || *rows[0].Field != "name" || *rows[1].Field != "assignees" || !reflect.DeepEqual(rows[2], moved) {
		t.Errorf("UpdateActivity() = %+v, want the name's, the assignee's, then the cycle's", rows)
	}
	if got := CreationJoins(nil); got != nil {
		t.Errorf("CreationJoins(nil) = %+v, want none", got)
	}
	if got := CreationJoins(&sprint); !reflect.DeepEqual(got, []Activity{joined}) {
		t.Errorf("CreationJoins() = %+v, want %+v", got, []Activity{joined})
	}
}

// skip_activity goes with description_html alone, so not with the cycle,
// given or cleared, which the description's own save would not write.
func TestSkipActivityRefusesTheCycle(t *testing.T) {
	html, skip, cycle := "<p>Fix</p>", true, uuid.NewV7()
	for _, p := range []IssuePatch{{DescriptionHTML: &html, SkipActivity: &skip, SetCycleID: true, CycleID: &cycle},
		{DescriptionHTML: &html, SkipActivity: &skip, SetCycleID: true}} {
		_, err := CheckIssuePatch(p)
		var e *shared.Error
		if !errors.As(err, &e) || len(e.Fields) != 1 || e.Fields[0].Field != "skip_activity" {
			t.Errorf("CheckIssuePatch(%+v) = %v; want skip_activity refused", p, err)
		}
	}
	if got, err := CheckIssuePatch(IssuePatch{SetCycleID: true}); err != nil || !got.SetCycleID {
		t.Errorf("CheckIssuePatch(cycle cleared) = %+v, %v; want it as given", got, err)
	}
}

// A cycle that a write may not join is the 422's problem at cycle_id.
func TestCycleNotAllowed(t *testing.T) {
	want := shared.FieldError{Field: "cycle_id", Code: shared.FieldNotAllowed, Message: "must be a cycle of the project"}
	if got := CycleNotAllowed(); got != want {
		t.Errorf("CycleNotAllowed() = %+v, want %+v", got, want)
	}
}
