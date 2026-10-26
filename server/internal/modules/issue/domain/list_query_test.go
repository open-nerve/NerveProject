package domain

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestParseListQuery(t *testing.T) {
	actor := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	other := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	t.Run("defaults", func(t *testing.T) {
		q, err := ParseListQuery(nil, ProjectList, actor)
		if err != nil || q.Order != "-created_at" || q.Limit != 50 || !q.Filters.SubIssues || q.Filters.Archived {
			t.Fatalf("defaults = %+v, %v", q, err)
		}
	})
	t.Run("CSV alternatives preserve independent fields and caller me", func(t *testing.T) {
		q, err := ParseListQuery(map[string][]string{
			"state_id": {other.String()}, "state_group": {"started,completed"}, "priority": {"urgent,none"},
			"label_id": {other.String() + ",none"}, "assignee_id": {"me,none," + other.String()},
			"created_by_id": {"me"}, "mention_id": {"me"}, "subscriber_id": {"me"},
			"sub_issue": {"false"}, "archived": {"true"}, "limit": {"100"}, "order_by": {"priority"},
		}, ProjectList, actor)
		if err != nil {
			t.Fatal(err)
		}
		want := ListFilters{StateIDs: []uuid.UUID{other}, StateGroups: []StateGroup{"started", "completed"}, Priorities: []Priority{"urgent", "none"}, Labels: IDFilter{IDs: []uuid.UUID{other}, None: true}, Assignees: IDFilter{IDs: []uuid.UUID{actor, other}, None: true}, CreatorIDs: []uuid.UUID{actor}, MentionIDs: []uuid.UUID{actor}, SubscriberIDs: []uuid.UUID{actor}, Archived: true}
		if !reflect.DeepEqual(q.Filters, want) || q.Limit != 100 || q.Order != "priority" {
			t.Fatalf("query = %+v; filters want %+v", q, want)
		}
	})
	t.Run("member relation and project filter", func(t *testing.T) {
		for _, relation := range []string{"assigned", "created", "subscribed"} {
			q, err := ParseListQuery(map[string][]string{"relation": {relation}, "project_id": {other.String()}, "group_by": {"project_id"}}, MemberList, actor)
			if err != nil || q.Relation != relation || !reflect.DeepEqual(q.Filters.ProjectIDs, []uuid.UUID{other}) {
				t.Fatalf("query=%+v err=%v", q, err)
			}
		}
	})
	t.Run("repeated dates retain closed independent conditions even disjoint", func(t *testing.T) {
		q, err := ParseListQuery(map[string][]string{"start_date": {"2026-01-01..2026-01-03", "2026-02-01.."}, "target_date": {"..9999-12-31", "0001-01-01"}}, ProjectList, actor)
		if err != nil {
			t.Fatal(err)
		}
		if len(q.Filters.Start) != 2 || q.Filters.Start[0].From.String() != "2026-01-01" || q.Filters.Start[0].To.String() != "2026-01-03" || q.Filters.Start[1].From.String() != "2026-02-01" || q.Filters.Start[1].To != nil {
			t.Fatalf("start=%+v", q.Filters.Start)
		}
		if len(q.Filters.Target) != 2 || q.Filters.Target[0].From != nil || q.Filters.Target[0].To.String() != "9999-12-31" || q.Filters.Target[1].From.String() != "0001-01-01" || q.Filters.Target[1].To.String() != "0001-01-01" {
			t.Fatalf("target=%+v", q.Filters.Target)
		}
	})
	t.Run("date none", func(t *testing.T) {
		q, err := ParseListQuery(map[string][]string{"target_date": {"none"}}, ProjectList, actor)
		if err != nil || len(q.Filters.Target) != 1 || !q.Filters.Target[0].None {
			t.Fatalf("query=%+v err=%v", q, err)
		}
	})
	t.Run("all nondate parameters reject repetition and empty value lists", func(t *testing.T) {
		for _, field := range []string{"state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "project_id", "sub_issue", "archived", "group_by", "sub_group_by", "group_key", "sub_group_key", "order_by", "relation", "limit", "cursor"} {
			for _, values := range [][]string{{"x", "x"}, {}} {
				scope := ProjectList
				raw := map[string][]string{field: values}
				if field == "project_id" || field == "relation" {
					scope = MemberList
					if field != "relation" {
						raw["relation"] = []string{"assigned"}
					}
				}
				_, err := ParseListQuery(raw, scope, actor)
				assertListProblem(t, err, shared.KindInvalid, field, shared.FieldNotAllowed)
			}
		}
	})
	t.Run("invalid filters and controls", func(t *testing.T) {
		cases := []struct{ field, value string }{
			{"state_id", "bad"}, {"state_id", "none"}, {"state_id", other.String() + ","}, {"priority", ",high"}, {"priority", "high,,low"}, {"priority", "HIGH"}, {"state_group", "triage"},
			{"label_id", "me"}, {"created_by_id", "none"}, {"mention_id", "none"}, {"subscriber_id", "none"}, {"assignee_id", " me"},
			{"start_date", "2026-02-30"}, {"start_date", "2026-03-01..2026-02-01"}, {"target_date", ".."}, {"target_date", "2026-01-01,2026-01-02"},
			{"sub_issue", "1"}, {"archived", ""}, {"cycle_id", "x"}, {"module_id", "x"}, {"link_count", "1"}, {"group_by", "state_name"}, {"order_by", "-sort_order"}, {"order_by", "link_count"}, {"order_by", "+created_at"}, {"order_by", ""}, {"relation", "assigned"},
		}
		for _, tc := range cases {
			t.Run(tc.field+"/"+tc.value, func(t *testing.T) {
				_, err := ParseListQuery(map[string][]string{tc.field: {tc.value}}, ProjectList, actor)
				assertListProblem(t, err, shared.KindInvalid, tc.field, shared.FieldNotAllowed)
			})
		}
		for _, field := range []string{"start_date", "target_date"} {
			for _, values := range [][]string{{"none", "none"}, {"none", "2026-01-01"}, {"2026-01-01", "none"}, {""}, {}} {
				_, err := ParseListQuery(map[string][]string{field: values}, ProjectList, actor)
				assertListProblem(t, err, shared.KindInvalid, field, shared.FieldNotAllowed)
			}
		}
	})
	t.Run("member excludes project capabilities", func(t *testing.T) {
		for _, field := range []string{"state_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "archived", "sub_group_by"} {
			_, err := ParseListQuery(map[string][]string{"relation": {"assigned"}, field: {other.String()}}, MemberList, actor)
			assertListProblem(t, err, shared.KindInvalid, field, shared.FieldNotAllowed)
		}
		_, err := ParseListQuery(map[string][]string{"relation": {"unknown"}}, MemberList, actor)
		assertListProblem(t, err, shared.KindInvalid, "relation", shared.FieldNotAllowed)
		_, err = ParseListQuery(nil, MemberList, actor)
		assertListProblem(t, err, shared.KindBadRequest, "relation", shared.FieldRequired)
		_, err = ParseListQuery(nil, ListScope(0), actor)
		if err == nil {
			t.Fatal("unknown scope accepted")
		}
	})
	t.Run("limit delegates range and preserves binding errors", func(t *testing.T) {
		for _, value := range []string{"0", "101", "-1"} {
			_, err := ParseListQuery(map[string][]string{"limit": {value}}, ProjectList, actor)
			assertListProblem(t, err, shared.KindInvalid, "limit", shared.FieldOutOfRange)
		}
		for _, value := range []string{"x", "1.5", "", "999999999999999999999999999"} {
			_, err := ParseListQuery(map[string][]string{"limit": {value}}, ProjectList, actor)
			assertListProblem(t, err, shared.KindBadRequest, "limit", shared.FieldInvalidFormat)
		}
		q, err := ParseListQuery(map[string][]string{"limit": {"1"}}, ProjectList, actor)
		if err != nil || q.Limit != 1 {
			t.Fatalf("query=%+v err=%v", q, err)
		}
	})
}

func assertListProblem(t *testing.T, err error, kind shared.Kind, field, code string) {
	t.Helper()
	var problem *shared.Error
	if !errors.As(err, &problem) || problem.Kind != kind || len(problem.Fields) != 1 || problem.Fields[0].Field != field || problem.Fields[0].Code != code {
		t.Fatalf("error=%#v, want kind=%v %s/%s", err, kind, field, code)
	}
}
