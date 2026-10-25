package postgresadapter

import (
	"encoding/json"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// This is the production Store mapper, not a fake Rows receiver. Dropping
// adapter normalization must fail independently of SQL's NULL defense.
func TestListParameterMapping(t *testing.T) {
	in := app.ListRead{Query: domain.ListQuery{Order: "sort_order", Limit: 1}}
	p, err := flatParameters(in)
	if err != nil {
		t.Fatal(err)
	}
	value := reflect.ValueOf(p)
	for n := 0; n < value.NumField(); n++ {
		if f := value.Field(n); f.Kind() == reflect.Slice && f.IsNil() {
			t.Errorf("nil sqlc array: %s", value.Type().Field(n).Name)
		}
	}
	a, b := uuid.NewV7(), uuid.NewV7()
	in.Directories = app.ListDirectories{Projects: []app.VisibleProject{{ID: b, OwnOnly: true}, {ID: a}}, States: []app.ListState{{ID: b, Group: "started", Key: "z"}, {ID: a, Group: "backlog", Key: "a"}}, Labels: []app.ListName{{ID: b, Key: "z"}, {ID: a, Key: "a"}}, Assignees: []app.ListName{{ID: b, Key: "b"}, {ID: a, Key: "a"}}}
	p, err = flatParameters(in)
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range []struct{ got, want any }{{p.ProjectIds, []uuid.UUID{b, a}}, {p.ProjectOwn, []bool{true, false}}, {p.StateIds, []uuid.UUID{b, a}}, {p.StateGroups, []string{"started", "backlog"}}, {p.StateKeys, []string{"z", "a"}}, {p.LabelIds, []uuid.UUID{b, a}}, {p.LabelKeys, []string{"z", "a"}}, {p.AssigneeIds, []uuid.UUID{b, a}}, {p.AssigneeKeys, []string{"b", "a"}}} {
		if !reflect.DeepEqual(pair.got, pair.want) {
			t.Errorf("paired array %v; want %v", pair.got, pair.want)
		}
	}
}

func TestListParameterTargetAndCursor(t *testing.T) {
	actor, target := uuid.NewV7(), uuid.NewV7()
	in := app.ListRead{ActorID: actor, TargetMemberID: &target, Query: domain.ListQuery{Order: "created_at", Relation: "assigned", Limit: 1}}
	p, err := flatParameters(in)
	if err != nil || p.ActorID != actor || p.RelationUserID == nil || *p.RelationUserID != target {
		t.Fatalf("caller/target parameters: %+v, %v", p, err)
	}
	for _, cursor := range []domain.ListCursor{
		{Order: "updated_at", Key: json.RawMessage(`"2026-01-01T00:00:00.000000Z"`), ID: target},
		{Order: "created_at", Key: json.RawMessage(`1`), ID: target},
	} {
		in.Query.Cursor = &cursor
		if _, err := flatParameters(in); err == nil {
			t.Fatalf("invalid internal cursor accepted: %+v", cursor)
		}
	}
}
