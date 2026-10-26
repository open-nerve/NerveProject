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

func TestIssueGroupParameters(t *testing.T) {
	actor, target := uuid.NewV7(), uuid.NewV7()
	key := "none"
	in := app.ListRead{ActorID: actor, TargetMemberID: &target, Query: domain.ListQuery{Order: "sort_order", Relation: "assigned", Limit: 1, GroupBy: "label_id", GroupKey: &key}}
	p, err := groupParameters(in)
	if err != nil {
		t.Fatal(err)
	}
	if p.ActorID != actor || p.RelationUserID == nil || *p.RelationUserID != target || p.GroupKey == nil || *p.GroupKey != key || p.GroupBy != "label_id" {
		t.Fatalf("group parameters %+v", p)
	}
	value := reflect.ValueOf(p)
	for n := 0; n < value.NumField(); n++ {
		if f := value.Field(n); f.Kind() == reflect.Slice && f.IsNil() {
			t.Errorf("nil group sqlc array %s", value.Type().Field(n).Name)
		}
	}
	for _, bad := range []domain.ListQuery{
		{Order: "sort_order", Limit: 1},
		{Order: "sort_order", Limit: 1, GroupBy: "unknown"},
		{Order: "sort_order", Limit: 1, GroupBy: "state_id", GroupKey: &key},
		{Order: "sort_order", Limit: 1, GroupBy: "label_id", SubGroupBy: "assignee_id"},
		{Order: "sort_order", Limit: 1, GroupBy: "label_id", Cursor: &domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: actor}},
	} {
		in.Query = bad
		in.Query.Relation = "assigned"
		if _, err := groupParameters(in); err == nil {
			t.Fatalf("accepted invalid group query %+v", bad)
		}
	}
}

func TestIssueSubgroupsParameters(t *testing.T) {
	actor, target := uuid.NewV7(), uuid.NewV7()
	key := "none"
	in := app.ListRead{ActorID: actor, TargetMemberID: &target, Query: domain.ListQuery{Order: "sort_order", Relation: "assigned", Limit: 1, GroupBy: "label_id", GroupKey: &key, SubGroupBy: "assignee_id", SubGroupKey: &key, Cursor: &domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: actor}}}
	p, err := subgroupParameters(in)
	if err != nil {
		t.Fatal(err)
	}
	if p.ActorID != actor || p.RelationUserID == nil || *p.RelationUserID != target || p.GroupBy != "label_id" || p.SubGroupBy != "assignee_id" || p.GroupKey == nil || *p.GroupKey != key || p.SubGroupKey == nil || *p.SubGroupKey != key || !p.HasCursor || p.CursorNum == nil || *p.CursorNum != 4607182418800017408 || p.PageSize != 2 {
		t.Fatalf("cell parameters %+v", p)
	}
	for name, isNil := range map[string]bool{
		"projects": p.ProjectIds == nil, "own": p.ProjectOwn == nil, "states": p.StateIds == nil, "state groups": p.StateGroups == nil, "state keys": p.StateKeys == nil,
		"labels": p.LabelIds == nil, "label keys": p.LabelKeys == nil, "assignees": p.AssigneeIds == nil, "assignee keys": p.AssigneeKeys == nil,
		"filter projects": p.FilterProjects == nil, "filter states": p.FilterStates == nil, "filter groups": p.FilterGroups == nil, "priorities": p.FilterPriorities == nil,
		"creators": p.FilterCreators == nil, "mentions": p.FilterMentions == nil, "subscribers": p.FilterSubscribers == nil, "filter labels": p.FilterLabels == nil, "filter assignees": p.FilterAssignees == nil,
		"start none": p.StartNone == nil, "start from": p.StartFrom == nil, "start to": p.StartTo == nil, "target none": p.TargetNone == nil, "target from": p.TargetFrom == nil, "target to": p.TargetTo == nil,
	} {
		if isNil {
			t.Errorf("nil cell sqlc array: %s", name)
		}
	}
	for _, bad := range []domain.ListQuery{
		{GroupBy: "label_id"}, {SubGroupBy: "assignee_id"}, {GroupBy: "priority", SubGroupBy: "priority"},
		{GroupBy: "project_id", SubGroupBy: "priority"}, {GroupBy: "label_id", SubGroupBy: "project_id"},
		{GroupBy: "label_id", SubGroupBy: "assignee_id", GroupKey: &key},
		{GroupBy: "label_id", SubGroupBy: "assignee_id", SubGroupKey: &key},
		{GroupBy: "state_id", SubGroupBy: "assignee_id", GroupKey: &key, SubGroupKey: &key},
		{GroupBy: "label_id", SubGroupBy: "state_group", GroupKey: &key, SubGroupKey: &key},
		{GroupBy: "label_id", SubGroupBy: "assignee_id", Cursor: in.Query.Cursor},
	} {
		in.Query = bad
		in.Query.Order, in.Query.Relation, in.Query.Limit = "sort_order", "assigned", 1
		if _, err := subgroupParameters(in); err == nil {
			t.Fatalf("accepted invalid cell query %+v", bad)
		}
	}
}
