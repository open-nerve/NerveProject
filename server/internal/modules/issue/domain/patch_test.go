package domain

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Nothing given, a name of one character and of 255, each priority, a
// hundred accounts and labels, the two sets given empty, and the dates
// given out of order or cleared: accepted, and returned as stored: the name
// without its surrounding white space, a set given empty still given (it
// empties the work item's), a date cleared still given, a field not given
// still not given. The order of the dates is the work item's once merged,
// which the use case holds under the lock (DatesOutOfOrder).
func TestCheckIssuePatchAccepts(t *testing.T) {
	state, none := uuid.NewV7(), []uuid.UUID{}
	many, labels := ids(MaxAssignees), ids(MaxLabels)
	tests := []struct {
		name string
		in   IssuePatch
		want IssuePatch
	}{
		{"nothing", IssuePatch{}, IssuePatch{}},
		{"one character, trimmed", IssuePatch{Name: ptr("  x\t")}, IssuePatch{Name: ptr("x")}},
		{"255 characters", IssuePatch{Name: ptr(strings.Repeat("é", 255))}, IssuePatch{Name: ptr(strings.Repeat("é", 255))}},
		{"everything", IssuePatch{Name: ptr("Fix"), StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: &many, LabelIDs: &labels},
			IssuePatch{Name: ptr("Fix"), StateID: &state, Priority: ptr(PriorityUrgent), AssigneeIDs: &many, LabelIDs: &labels}},
		{"both sets empty", IssuePatch{AssigneeIDs: &none, LabelIDs: &none}, IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
		{"the dates out of order", IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)},
			IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)}},
		{"both dates cleared", IssuePatch{SetStartDate: true, SetTargetDate: true}, IssuePatch{SetStartDate: true, SetTargetDate: true}},
	}
	for _, p := range priorities {
		tests = append(tests, struct {
			name string
			in   IssuePatch
			want IssuePatch
		}{string(p), IssuePatch{Priority: ptr(p)}, IssuePatch{Priority: ptr(p)}})
	}
	for _, tt := range tests {
		if got, err := CheckIssuePatch(tt.in); err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CheckIssuePatch() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
	}
}

// A blank name, one of 256 characters, one with NUL, a priority outside the
// five, in another case or given empty, an account or a label named twice,
// a hundred and one of either: each refused, all of them in one 422 that
// names each by its field and place, as CheckNewIssue does (M4 design 3.7).
func TestCheckIssuePatchReportsEveryProblem(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	repeats, twice := []uuid.UUID{a, b, a}, []uuid.UUID{b, b}
	many := append(ids(MaxAssignees-1), a, a)
	priority := shared.FieldError{Field: "priority", Code: "invalid_format", Message: "must be one of urgent, high, medium, low, none"}
	tests := []struct {
		name string
		in   IssuePatch
		want []shared.FieldError
	}{
		{"blank", IssuePatch{Name: ptr(" \n ")}, []shared.FieldError{{Field: "name", Code: "too_short", Message: "must not be empty"}}},
		{"256 characters", IssuePatch{Name: ptr(strings.Repeat("x", 256))},
			[]shared.FieldError{{Field: "name", Code: "too_long", Message: "must be at most 255 characters"}}},
		{"NUL", IssuePatch{Name: ptr("a\x00b")}, []shared.FieldError{{Field: "name", Code: "invalid_format", Message: "must not contain a NUL character"}}},
		{"another case", IssuePatch{Priority: ptr[Priority]("Urgent")}, []shared.FieldError{priority}},
		{"given empty", IssuePatch{Priority: ptr[Priority]("")}, []shared.FieldError{priority}},
		{"repeats", IssuePatch{AssigneeIDs: &repeats, LabelIDs: &twice}, []shared.FieldError{
			{Field: "assignee_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "label_ids[1]", Code: "duplicate", Message: "is listed before"},
		}},
		{"too many, and everything else", IssuePatch{Name: ptr(""), Priority: ptr[Priority]("high!"), AssigneeIDs: &many, LabelIDs: &many},
			[]shared.FieldError{
				{Field: "name", Code: "too_short", Message: "must not be empty"},
				priority,
				{Field: "assignee_ids", Code: "too_long", Message: "must name at most 100"},
				{Field: "label_ids", Code: "too_long", Message: "must name at most 100"},
			}},
	}
	for _, tt := range tests {
		var e *shared.Error
		if _, err := CheckIssuePatch(tt.in); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckIssuePatch() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
}

// What a patch changes in a work item, Fix in Todo, high, from the 12th
// with no target date, under up, assigned to ann and ben, labelled Bug and
// UI (M4 design 3.7, 3.9): a field not given, or given as it is, changes
// nothing, a date the same day or none again too, the parent the same
// work item, or none again; a date given another day, or cleared, is the
// new one, and so is a parent given another work item, or none; a set
// given is the new set, whose difference with the work item's is added
// and removed, each in id order, whatever the order given; a set not given
// leaves the work item's as it is, its accounts and labels too.
func TestChangesOf(t *testing.T) {
	// In id order: ann, ben, cat, dan; bug, ui, ux.
	people, tags := ids(4), ids(3)
	ann, ben, cat, dan := people[0], people[1], people[2], people[3]
	bug, ui, ux := tags[0], tags[1], tags[2]
	todo, done, up, top := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	i := Issue{Name: "Fix", StateID: todo, Priority: PriorityHigh, StartDate: october(12), ParentID: ptr(up),
		AssigneeIDs: []uuid.UUID{ann, ben}, LabelIDs: []uuid.UUID{bug, ui}}
	set := func(ids ...uuid.UUID) *[]uuid.UUID { return &ids }
	tests := []struct {
		name string
		p    IssuePatch
		want Changes
	}{
		{"nothing given", IssuePatch{}, Changes{}},
		{"each given as it is", IssuePatch{Name: ptr("Fix"), StateID: &todo, Priority: ptr(PriorityHigh), SetStartDate: true,
			StartDate: october(12), SetTargetDate: true, SetParentID: true, ParentID: ptr(up), AssigneeIDs: set(ben, ann),
			LabelIDs: set(ui, bug)}, Changes{}},
		{"another parent", IssuePatch{SetParentID: true, ParentID: &top}, Changes{Parent: &ParentChange{To: &top}}},
		{"the parent taken away", IssuePatch{SetParentID: true}, Changes{Parent: &ParentChange{}}},
		{"the name", IssuePatch{Name: ptr("Mend")}, Changes{Name: ptr("Mend")}},
		{"the priority", IssuePatch{Priority: ptr(PriorityNone)}, Changes{Priority: ptr(PriorityNone)}},
		{"the state", IssuePatch{StateID: &done}, Changes{StateID: &done}},
		{"the start date", IssuePatch{SetStartDate: true, StartDate: october(13)}, Changes{StartDate: &DateChange{To: october(13)}}},
		{"the start date cleared", IssuePatch{SetStartDate: true}, Changes{StartDate: &DateChange{}}},
		{"a target date", IssuePatch{SetTargetDate: true, TargetDate: october(19)}, Changes{TargetDate: &DateChange{To: october(19)}}},
		{"accounts added and removed, out of order", IssuePatch{AssigneeIDs: set(dan, ben, cat)},
			Changes{Assignees: Diff{Added: []uuid.UUID{cat, dan}, Removed: []uuid.UUID{ann}}}},
		{"every account removed", IssuePatch{AssigneeIDs: set()}, Changes{Assignees: Diff{Removed: []uuid.UUID{ann, ben}}}},
		{"labels added and removed, out of order", IssuePatch{LabelIDs: set(ux, bug)},
			Changes{Labels: Diff{Added: []uuid.UUID{ux}, Removed: []uuid.UUID{ui}}}},
		{"every label removed", IssuePatch{LabelIDs: set()}, Changes{Labels: Diff{Removed: []uuid.UUID{bug, ui}}}},
	}
	for _, tt := range tests {
		if got := ChangesOf(i, tt.p); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: ChangesOf() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
	orphan := Issue{Name: "Fix", StateID: todo}
	if got := ChangesOf(orphan, IssuePatch{SetParentID: true}); !reflect.DeepEqual(got, Changes{}) {
		t.Errorf("no parent taken away from a work item under none: ChangesOf() = %+v, want nothing", got)
	}
	if got, want := ChangesOf(orphan, IssuePatch{SetParentID: true, ParentID: &up}), (Changes{Parent: &ParentChange{To: &up}}); !reflect.DeepEqual(got,
		want) {
		t.Errorf("a parent for a work item under none: ChangesOf() = %+v, want %+v", got, want)
	}
}

// Of the accounts asked for, only those added are checked (M4 design 3.6):
// an account added who is the project's guest, or no active member of it,
// is a problem at its index in what is asked; one the work item keeps is
// none, whatever his role, and nor is one added who may be assigned.
func TestUnassignable(t *testing.T) {
	people := ids(4) // in id order: ann, ben, gus, out
	ann, ben, gus, out := people[0], people[1], people[2], people[3]
	roles := map[uuid.UUID]shared.Role{ann: shared.RoleAdmin, ben: shared.RoleMember, gus: shared.RoleGuest}
	refused := func(i int) shared.FieldError {
		return shared.FieldError{Field: fmt.Sprintf("assignee_ids[%d]", i), Code: shared.FieldNotAllowed,
			Message: "must be an active member of the project who is not its guest"}
	}
	tests := []struct {
		name         string
		asked, added []uuid.UUID
		want         []shared.FieldError
	}{
		{"none added", []uuid.UUID{gus, out}, nil, nil},
		{"an admin and a member added", []uuid.UUID{ben, ann}, []uuid.UUID{ann, ben}, nil},
		{"a guest and one who left added, both kept too", []uuid.UUID{ann, out, gus, ben}, []uuid.UUID{gus, out},
			[]shared.FieldError{refused(1), refused(2)}},
		{"a guest kept, one who left added", []uuid.UUID{gus, out}, []uuid.UUID{out}, []shared.FieldError{refused(1)}},
	}
	for _, tt := range tests {
		if got := Unassignable(tt.asked, tt.added, roles); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: Unassignable() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// Changes change nothing only when each of them is empty: any one of them
// alone is a change.
func TestChangesNone(t *testing.T) {
	id := uuid.NewV7()
	one := []uuid.UUID{id}
	if !(Changes{}).None() {
		t.Error("Changes{}.None() = false, want true")
	}
	for name, c := range map[string]Changes{
		"the name": {Name: ptr("x")}, "the priority": {Priority: ptr(PriorityLow)}, "the state": {StateID: &id},
		"the start date cleared": {StartDate: &DateChange{}}, "a target date": {TargetDate: &DateChange{To: october(19)}},
		"the parent taken away": {Parent: &ParentChange{}}, "a parent": {Parent: &ParentChange{To: &id}},
		"an account added": {Assignees: Diff{Added: one}}, "an account removed": {Assignees: Diff{Removed: one}},
		"a label added": {Labels: Diff{Added: one}}, "a label removed": {Labels: Diff{Removed: one}},
	} {
		if c.None() {
			t.Errorf("%s: None() = true, want false", name)
		}
	}
}
