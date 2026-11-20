package domain

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The workspace's cross-project list (M7 P3a spec 2.2, R2): its twelve
// filter words, not a cycle's, a module's, the archive's or a member's
// relation; and flat only, no group word on it. The table catches a word
// admitted or dropped; the existing tables hold the scopes as columns and
// stay as they are.
func TestWorkspaceListVocabulary(t *testing.T) {
	for _, tc := range []struct {
		field  string
		filter bool
	}{
		{"project_id", true}, {"state_id", true}, {"state_group", true}, {"priority", true}, {"label_id", true},
		{"assignee_id", true}, {"created_by_id", true}, {"mention_id", true}, {"subscriber_id", true},
		{"start_date", true}, {"target_date", true}, {"sub_issue", true},
		{"cycle_id", false}, {"module_id", false}, {"archived", false}, {"relation", false},
		{"group_by", false}, {"sub_group_by", false}, {"group_key", false}, {"sub_group_key", false},
		{"order_by", false}, {"cursor", false}, {"limit", false}, {"status", false}, {"created_at", false},
		{"creator", false}, {"cycle_ids", false}, {"module_ids", false}, {"", false},
	} {
		t.Run("filter/"+tc.field, func(t *testing.T) {
			if got := ListFilterAllowed(tc.field, WorkspaceList); got != tc.filter {
				t.Errorf("workspace filter %q = %v, want %v", tc.field, got, tc.filter)
			}
		})
	}
	// Every group a project's or a member's list has, and none besides.
	for _, group := range []ListGroup{"state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "cycle_id", "module_id", "target_date", "project_id", ""} {
		t.Run("group/"+string(group), func(t *testing.T) {
			if ListGroupAllowed(group, WorkspaceList) || ListSubGroupAllowed(group, WorkspaceList) {
				t.Errorf("the workspace list is flat; it groups by %q", group)
			}
		})
	}
}

func TestParseWorkspaceListQuery(t *testing.T) {
	actor := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	other := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	t.Run("defaults", func(t *testing.T) {
		q, err := ParseListQuery(nil, WorkspaceList, actor)
		want := ListQuery{Order: "-created_at", Limit: 50, Filters: ListFilters{SubIssues: true}}
		if err != nil || !reflect.DeepEqual(q, want) {
			t.Fatalf("defaults = %+v, %v; want %+v", q, err, want)
		}
	})
	t.Run("every filter word, me as the caller", func(t *testing.T) {
		q, err := ParseListQuery(map[string][]string{
			"project_id": {other.String() + "," + actor.String()}, "state_id": {other.String()},
			"state_group": {"started,completed"}, "priority": {"urgent,none"}, "label_id": {other.String() + ",none"},
			"assignee_id": {"me,none," + other.String()}, "created_by_id": {"me"}, "mention_id": {"me," + other.String()},
			"subscriber_id": {"me"}, "start_date": {"2026-01-01..", "..2026-02-01"}, "target_date": {"none"},
			"sub_issue": {"false"}, "order_by": {"priority"}, "limit": {"100"},
		}, WorkspaceList, actor)
		if err != nil {
			t.Fatal(err)
		}
		want := ListFilters{ProjectIDs: []uuid.UUID{other, actor}, StateIDs: []uuid.UUID{other},
			StateGroups: []StateGroup{"started", "completed"}, Priorities: []Priority{"urgent", "none"},
			Labels: IDFilter{IDs: []uuid.UUID{other}, None: true}, Assignees: IDFilter{IDs: []uuid.UUID{actor, other}, None: true},
			CreatorIDs: []uuid.UUID{actor}, MentionIDs: []uuid.UUID{actor, other}, SubscriberIDs: []uuid.UUID{actor}}
		if len(q.Filters.Start) != 2 || len(q.Filters.Target) != 1 || !q.Filters.Target[0].None {
			t.Fatalf("dates = %+v, %+v", q.Filters.Start, q.Filters.Target)
		}
		q.Filters.Start, q.Filters.Target = nil, nil
		if !reflect.DeepEqual(q.Filters, want) || q.Order != "priority" || q.Limit != 100 || q.Relation != "" || q.GroupBy != "" || q.GroupKey != nil {
			t.Fatalf("query = %+v; filters want %+v", q, want)
		}
	})
	// The orders are every page's (R3), the cycles' and the modules' names
	// too, and so is the cursor's envelope.
	t.Run("orders and cursor", func(t *testing.T) {
		for _, order := range []ListOrder{"sort_order", "cycle_name", "-module_name", "label_name", "-attachment_count"} {
			q, err := ParseListQuery(map[string][]string{"order_by": {string(order)}}, WorkspaceList, actor)
			if err != nil || q.Order != order {
				t.Errorf("order %q: %+v, %v", order, q, err)
			}
		}
		cursor, err := EncodeListCursor(ListCursor{Order: "sort_order", Key: []byte("1"), ID: other})
		if err != nil {
			t.Fatal(err)
		}
		q, err := ParseListQuery(map[string][]string{"order_by": {"sort_order"}, "cursor": {cursor}}, WorkspaceList, actor)
		if err != nil || q.Cursor == nil || q.Cursor.ID != other {
			t.Fatalf("cursor: %+v, %v", q, err)
		}
	})
	// Not a cycle's, a module's or the archive's word (§15 D4), not a
	// member's relation, and flat only: each group word answers 422 by
	// itself, never a grouped or a continued page.
	t.Run("refused words", func(t *testing.T) {
		for _, tc := range []struct{ field, value string }{
			{"cycle_id", other.String()}, {"cycle_id", "none"}, {"module_id", other.String()}, {"module_id", "none"},
			{"archived", "false"}, {"archived", "true"},
			{"relation", "assigned"}, {"relation", "created"}, {"relation", "subscribed"},
			{"group_by", "priority"}, {"group_by", "project_id"}, {"group_by", "state_group"},
			{"sub_group_by", "priority"}, {"group_key", "high"}, {"group_key", "none"}, {"sub_group_key", "high"},
			{"order_by", "sequence_id"}, {"status", "pending"}, {"target_member_id", other.String()}, {"unknown", "x"},
		} {
			t.Run(tc.field+"/"+tc.value, func(t *testing.T) {
				_, err := ParseListQuery(map[string][]string{tc.field: {tc.value}}, WorkspaceList, actor)
				assertListProblem(t, err, shared.KindInvalid, tc.field, shared.FieldNotAllowed)
			})
		}
		// A group and its key together: the group word answers first.
		_, err := ParseListQuery(map[string][]string{"group_by": {"priority"}, "group_key": {"high"}}, WorkspaceList, actor)
		assertListProblem(t, err, shared.KindInvalid, "group_by", shared.FieldNotAllowed)
	})
	t.Run("repeated non-date words", func(t *testing.T) {
		for _, field := range []string{"project_id", "state_id", "state_group", "priority", "label_id", "assignee_id", "created_by_id", "mention_id", "subscriber_id", "sub_issue", "order_by", "limit", "cursor"} {
			for _, values := range [][]string{{"x", "x"}, {}} {
				_, err := ParseListQuery(map[string][]string{field: values}, WorkspaceList, actor)
				assertListProblem(t, err, shared.KindInvalid, field, shared.FieldNotAllowed)
			}
		}
	})
}
