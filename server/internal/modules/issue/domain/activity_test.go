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
