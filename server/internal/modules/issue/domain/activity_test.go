package domain

import (
	"reflect"
	"testing"
	"uuid"
)

// A creation writes row 1, the creation, alone with no field, value or
// id; then row 2 for each account it is assigned to, in the order given:
// assignees, updated, from "" to his display name, his id the new
// identifier (M4 design 3.13). Plane's sentences go with them, its brand
// left out.
func TestCreationActivity(t *testing.T) {
	ann, ben := uuid.NewV7(), uuid.NewV7()
	created := Activity{Verb: "created", Comment: "created the work item"}
	assigned := func(id uuid.UUID, name string) Activity {
		field, empty := "assignees", ""
		return Activity{Field: &field, Verb: "updated", OldValue: &empty, NewValue: &name, NewIdentifier: &id, Comment: "added assignee "}
	}
	tests := []struct {
		name      string
		assignees []Assignee
		want      []Activity
	}{
		{"no assignee", nil, []Activity{created}},
		{"two", []Assignee{{ID: ben, DisplayName: "ben"}, {ID: ann, DisplayName: "Ann L."}},
			[]Activity{created, assigned(ben, "ben"), assigned(ann, "Ann L.")}},
	}
	for _, tt := range tests {
		if got := CreationActivity(tt.assignees); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: CreationActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

// An update writes a row for each field it changes, in the order of M4
// design 3.13's table, whatever else it changes: the name (row 3), the
// priority (row 6), the state by its names and ids (row 7), each label
// added and removed by its name and id (rows 10, 11), each account
// assigned and unassigned by his display name and id (rows 12, 13), each
// list in the order given; Plane's sentences go with them. A field left as
// it is writes none.
func TestUpdateActivity(t *testing.T) {
	todo, done, bug, ui, ann, ben := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	row := func(field string, oldValue, newValue string, oldID, newID *uuid.UUID, comment string) Activity {
		return Activity{Field: &field, Verb: "updated", OldValue: &oldValue, NewValue: &newValue, OldIdentifier: oldID, NewIdentifier: newID,
			Comment: comment}
	}
	name := row("name", "Fix", "Mend", nil, nil, "updated the name to")
	priority := row("priority", "high", "none", nil, nil, "updated the priority to")
	state := row("state", "Todo", "Done", &todo, &done, "updated the state to")
	labelled := func(id uuid.UUID, n string) Activity { return row("labels", "", n, nil, &id, "added label ") }
	unlabelled := func(id uuid.UUID, n string) Activity { return row("labels", n, "", &id, nil, "removed label ") }
	assigned := func(id uuid.UUID, n string) Activity { return row("assignees", "", n, nil, &id, "added assignee ") }
	unassigned := func(id uuid.UUID, n string) Activity { return row("assignees", n, "", &id, nil, "removed assignee ") }
	tests := []struct {
		name string
		u    Update
		want []Activity
	}{
		{"nothing", Update{}, nil},
		{"everything", Update{
			AssigneesRemoved: []Assignee{{ID: ben, DisplayName: "ben"}}, AssigneesAdded: []Assignee{{ID: ann, DisplayName: "Ann L."}},
			LabelsRemoved: []Named{{ID: ui, Name: "UI"}}, LabelsAdded: []Named{{ID: bug, Name: "Bug"}},
			State:    &Changed[Named]{Old: Named{ID: todo, Name: "Todo"}, New: Named{ID: done, Name: "Done"}},
			Priority: &Changed[Priority]{Old: PriorityHigh, New: PriorityNone}, Name: &Changed[string]{Old: "Fix", New: "Mend"},
		}, []Activity{name, priority, state, labelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ann, "Ann L."), unassigned(ben, "ben")}},
		{"the state alone", Update{State: &Changed[Named]{Old: Named{ID: todo, Name: "Todo"}, New: Named{ID: done, Name: "Done"}}},
			[]Activity{state}},
		{"two labels and two accounts each way, in the order given", Update{
			LabelsAdded: []Named{{ID: ui, Name: "UI"}, {ID: bug, Name: "Bug"}}, LabelsRemoved: []Named{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
			AssigneesAdded:   []Assignee{{ID: ben, DisplayName: "ben"}, {ID: ann, DisplayName: "ann"}},
			AssigneesRemoved: []Assignee{{ID: ann, DisplayName: "ann"}, {ID: ben, DisplayName: "ben"}},
		}, []Activity{labelled(ui, "UI"), labelled(bug, "Bug"), unlabelled(bug, "Bug"), unlabelled(ui, "UI"), assigned(ben, "ben"),
			assigned(ann, "ann"), unassigned(ann, "ann"), unassigned(ben, "ben")}},
	}
	for _, tt := range tests {
		if got := UpdateActivity(tt.u); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: UpdateActivity() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
