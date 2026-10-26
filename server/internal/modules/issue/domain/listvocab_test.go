package domain

import "testing"

// These tables catch accidental admission of future words, project-only
// capabilities on member pages, and me/none on the wrong fields.
func TestListVocabulary(t *testing.T) {
	t.Run("orders", func(t *testing.T) {
		for _, order := range []ListOrder{
			"sort_order", "created_at", "-created_at", "updated_at", "-updated_at",
			"start_date", "-start_date", "target_date", "-target_date", "priority", "-priority",
			"state_name", "-state_name", "label_name", "-label_name", "assignee_name", "-assignee_name",
			"sub_issues_count", "-sub_issues_count",
		} {
			t.Run(string(order), func(t *testing.T) {
				if !ListOrderAllowed(order) {
					t.Errorf("order %q must be allowed", order)
				}
			})
		}
		for _, order := range []ListOrder{"", "-sort_order", "link_count", "-link_count", "+link_count", "+created_at", "--priority", "Created_at", "created_at ", "cycle_id"} {
			t.Run("reject_"+string(order), func(t *testing.T) {
				if ListOrderAllowed(order) {
					t.Errorf("order %q must be rejected", order)
				}
			})
		}
	})
	t.Run("filters", func(t *testing.T) {
		for _, tc := range []struct {
			field           string
			project, member bool
		}{
			{"state_id", true, false}, {"state_group", true, true}, {"priority", true, true},
			{"label_id", true, true}, {"assignee_id", true, false}, {"created_by_id", true, false},
			{"mention_id", true, false}, {"subscriber_id", true, false}, {"project_id", false, true},
			{"start_date", true, true}, {"target_date", true, true}, {"sub_issue", true, true}, {"archived", true, false},
			{"creator", false, false}, {"subgroup", false, false}, {"relation", false, false},
			{"group_by", false, false}, {"sub_group_by", false, false}, {"order_by", false, false},
			{"cursor", false, false}, {"limit", false, false}, {"cycle_id", false, false},
			{"module_id", false, false}, {"link_count", false, false}, {"", false, false},
		} {
			t.Run(tc.field, func(t *testing.T) {
				if got := ListFilterAllowed(tc.field, ProjectList); got != tc.project {
					t.Errorf("project filter = %v, want %v", got, tc.project)
				}
				if got := ListFilterAllowed(tc.field, MemberList); got != tc.member {
					t.Errorf("member filter = %v, want %v", got, tc.member)
				}
				if ListFilterAllowed(tc.field, ListScope(0)) || ListFilterAllowed(tc.field, ListScope(255)) {
					t.Error("unknown scope must reject filters")
				}
			})
		}
	})
	t.Run("groups", func(t *testing.T) {
		for _, tc := range []struct {
			group           ListGroup
			project, member bool
		}{
			{"state_id", true, false}, {"state_group", true, true}, {"priority", true, true},
			{"label_id", true, true}, {"assignee_id", true, false}, {"created_by_id", true, false},
			{"target_date", true, true}, {"project_id", false, true},
			{"start_date", false, false}, {"mention_id", false, false}, {"subscriber_id", false, false},
			{"state_name", false, false}, {"label_name", false, false}, {"assignee_name", false, false},
			{"creator", false, false}, {"archived", false, false}, {"subgroup", false, false}, {"", false, false},
		} {
			t.Run(string(tc.group), func(t *testing.T) {
				if got := ListGroupAllowed(tc.group, ProjectList); got != tc.project {
					t.Errorf("project group = %v, want %v", got, tc.project)
				}
				if got := ListGroupAllowed(tc.group, MemberList); got != tc.member {
					t.Errorf("member group = %v, want %v", got, tc.member)
				}
				if got := ListSubGroupAllowed(tc.group, ProjectList); got != tc.project {
					t.Errorf("project subgroup = %v, want %v", got, tc.project)
				}
				if ListSubGroupAllowed(tc.group, MemberList) {
					t.Error("member page must reject subgroups")
				}
				if ListGroupAllowed(tc.group, ListScope(0)) || ListSubGroupAllowed(tc.group, ListScope(0)) || ListGroupAllowed(tc.group, ListScope(255)) || ListSubGroupAllowed(tc.group, ListScope(255)) {
					t.Error("unknown scope must reject groups and subgroups")
				}
			})
		}
	})
	t.Run("filter_atoms", func(t *testing.T) {
		id := "11111111-1111-4111-8111-111111111111"
		for _, tc := range []struct {
			field    string
			me, none bool
		}{
			{"state_id", false, false}, {"project_id", false, false}, {"label_id", false, true},
			{"assignee_id", true, true}, {"created_by_id", true, false},
			{"mention_id", true, false}, {"subscriber_id", true, false},
		} {
			t.Run(tc.field, func(t *testing.T) {
				for _, value := range []string{id, "00000000-0000-0000-0000-000000000000"} {
					if !ListFilterValueAllowed(tc.field, value) {
						t.Errorf("syntactically valid UUID %q must be accepted without an existence check", value)
					}
				}
				if got := ListFilterValueAllowed(tc.field, "me"); got != tc.me {
					t.Errorf("me = %v, want %v", got, tc.me)
				}
				if got := ListFilterValueAllowed(tc.field, "none"); got != tc.none {
					t.Errorf("none = %v, want %v", got, tc.none)
				}
				for _, value := range []string{"", "bad-uuid", id + "," + id, " " + id, "Me", "NONE"} {
					if ListFilterValueAllowed(tc.field, value) {
						t.Errorf("atom %q must be rejected", value)
					}
				}
			})
		}
		for _, field := range []string{"priority", "state_group", "sub_issue", "archived", "relation", "future"} {
			t.Run(field, func(t *testing.T) {
				var accepted []string
				switch field {
				case "priority":
					accepted = []string{"urgent", "high", "medium", "low", "none"}
				case "state_group":
					accepted = []string{"backlog", "unstarted", "started", "completed", "cancelled"}
				case "sub_issue", "archived":
					accepted = []string{"true", "false"}
				}
				for _, value := range accepted {
					if !ListFilterValueAllowed(field, value) {
						t.Errorf("atom %q must be accepted", value)
					}
				}
				for _, value := range []string{"", "triage", "me", "true,false", "Urgent", "1", "TRUE", "false ", id} {
					if ListFilterValueAllowed(field, value) {
						t.Errorf("atom %q must be rejected", value)
					}
				}
			})
		}
		for _, field := range []string{"start_date", "target_date"} {
			for _, value := range []string{"2026-10-25", "2026-01-01..2026-12-31", "..2026-10-25", "2026-10-25..", "none"} {
				if !ListFilterValueAllowed(field, value) {
					t.Errorf("%s atom %q must be accepted", field, value)
				}
			}
			for _, value := range []string{"..", "2026-12-31..2026-01-01", "2026-02-30", "2026-10-25,2026-10-26", "none..2026-10-25"} {
				if ListFilterValueAllowed(field, value) {
					t.Errorf("%s atom %q must be rejected", field, value)
				}
			}
		}
	})
	t.Run("group_keys", func(t *testing.T) {
		id := "11111111-1111-4111-8111-111111111111"
		for _, tc := range []struct {
			group ListGroup
			good  []string
			bad   []string
		}{
			{"state_id", []string{id}, []string{"none", "me", "bad"}},
			{"project_id", []string{id}, []string{"none", "me", "bad"}},
			{"label_id", []string{id, "none"}, []string{"me", "bad", id + "," + id}},
			{"assignee_id", []string{id, "none"}, []string{"me", "bad"}},
			{"created_by_id", []string{id, "none"}, []string{"me", "bad"}},
			{"priority", []string{"urgent", "high", "medium", "low", "none"}, []string{"me", "triage", id}},
			{"state_group", []string{"backlog", "unstarted", "started", "completed", "cancelled"}, []string{"none", "triage", id}},
			{"target_date", []string{"2026-10-25", "none", "9999-12-31"}, []string{"..2026-10-25", "2026-10-25..", "2026-10-25..2026-10-26", "2026-02-30"}},
			{"start_date", nil, []string{"2026-10-25", "none"}},
			{"state_name", nil, []string{"Todo", "none"}},
			{"mention_id", nil, []string{id, "none"}},
			{"", nil, []string{id, "none"}},
		} {
			t.Run(string(tc.group), func(t *testing.T) {
				for _, key := range tc.good {
					if !ListGroupKeyAllowed(tc.group, key) {
						t.Errorf("key %q must be accepted", key)
					}
				}
				for _, key := range append(tc.bad, "") {
					if ListGroupKeyAllowed(tc.group, key) {
						t.Errorf("key %q must be rejected", key)
					}
				}
			})
		}
	})
}
