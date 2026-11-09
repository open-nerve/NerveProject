package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"
)

// A project's list filters its work items by cycle and by module, each in
// the flat, grouped and subgrouped forms (M6 P3b spec 2.1, A1): the values of
// one field are alternatives, none meaning no undeleted membership, and
// different fields intersect; a membership left never counts, an archived
// cycle or module does, and an id is not looked up. Walt, Web's guest and
// acme's admin, sees only his own work items (OwnOnly), so his totals and his
// groups' count only those. a, alice's, labelled Bug, is in Sprint and Core;
// b, walt's, in Sprint, Core and UI, archived; c, walt's, in Old, archived,
// having left Sprint and Core; d, alice's, in UI, having left Core; e,
// walt's, in nothing, having left Sprint and UI. The member's list refuses
// both filters, as it refuses assignee_id, and the project's refuses either
// given twice, before the caller is known.
func TestIssueListPlanningFilters(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	priorities := map[uuid.UUID]string{}
	item := func(by, name, priority, more string) uuid.UUID {
		id := w.newWorkItemBy(t, by, w.web, fmt.Sprintf(`{"name":%q,"priority":%q%s}`, name, priority, more))
		priorities[id] = priority
		return id
	}
	a := item("alice", "A", "high", `,"label_ids":`+idArray(w.bug))
	b := item("walt", "B", "high", "")
	c := item("walt", "C", "none", "")
	d := item("alice", "D", "none", "")
	e := item("walt", "E", "low", "")
	sprint, old := w.seedPlanningTarget(t, "cycles", "Sprint"), w.seedPlanningTarget(t, "cycles", "Old")
	core, ui := w.seedPlanningTarget(t, "modules", "Core"), w.seedPlanningTarget(t, "modules", "UI")
	listSeedSQL(t, w, "UPDATE cycles SET archived_at = now() WHERE id = $1", old)
	listSeedSQL(t, w, "UPDATE modules SET archived_at = now() WHERE id = $1", ui)
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", a, sprint, false}, {"module_issues", "module_id", a, core, false},
		{"cycle_issues", "cycle_id", b, sprint, false}, {"module_issues", "module_id", b, core, false}, {"module_issues", "module_id", b, ui, false},
		{"cycle_issues", "cycle_id", c, sprint, true}, {"cycle_issues", "cycle_id", c, old, false}, {"module_issues", "module_id", c, core, true},
		{"module_issues", "module_id", d, core, true}, {"module_issues", "module_id", d, ui, false},
		{"cycle_issues", "cycle_id", e, sprint, true}, {"module_issues", "module_id", e, ui, true},
	} {
		w.seedMembership(t, row.table, row.column, row.issue, row.target, row.gone)
	}
	listed := func(t *testing.T, name, query string, want ...uuid.UUID) {
		t.Helper()
		inGroup := map[string]int64{}
		for _, id := range want {
			inGroup[priorities[id]]++
		}
		want = slices.SortedFunc(slices.Values(want), uuid.UUID.Compare)
		for _, form := range []string{"", "&group_by=priority", "&group_by=priority&sub_group_by=state_group"} {
			page := checkListEnvelope(t, w.send(t, name, http.MethodGet, path+"?"+query+form, "", http.StatusOK), form != "", strings.Contains(form, "sub_group"))
			var got []uuid.UUID
			for _, row := range page.Data {
				got = append(got, row.ID)
			}
			for _, g := range page.Groups {
				if g.Total != inGroup[g.Key] {
					t.Errorf("%s %s%s: group %s of %d; want %d", name, query, form, g.Key, g.Total, inGroup[g.Key])
				}
				for _, row := range g.Data {
					got = append(got, row.ID)
				}
				for _, cell := range g.Subgroups {
					if cell.Total != g.Total {
						t.Errorf("%s %s%s: cell %s/%s of %d; want %d", name, query, form, g.Key, cell.Key, cell.Total, g.Total)
					}
					for _, row := range cell.Data {
						got = append(got, row.ID)
					}
				}
			}
			for _, lane := range page.SubgroupTotals {
				if lane.Total != int64(len(want)) {
					t.Errorf("%s %s%s: lane %s of %d; want %d", name, query, form, lane.Key, lane.Total, len(want))
				}
			}
			slices.SortFunc(got, uuid.UUID.Compare)
			if page.Total != int64(len(want)) || !slices.Equal(got, want) {
				t.Errorf("%s %s%s: %d work items %v; want %v", name, query, form, page.Total, got, want)
			}
		}
	}
	for _, tc := range []struct {
		query       string
		alice, walt []uuid.UUID
	}{
		{"cycle_id=" + sprint.String(), []uuid.UUID{a, b}, []uuid.UUID{b}},
		{"cycle_id=" + old.String(), []uuid.UUID{c}, []uuid.UUID{c}},
		{"cycle_id=none", []uuid.UUID{d, e}, []uuid.UUID{e}},
		{"cycle_id=none," + sprint.String(), []uuid.UUID{a, b, d, e}, []uuid.UUID{b, e}},
		{"module_id=" + core.String(), []uuid.UUID{a, b}, []uuid.UUID{b}},
		{"module_id=" + ui.String(), []uuid.UUID{b, d}, []uuid.UUID{b}},
		{"module_id=none", []uuid.UUID{c, e}, []uuid.UUID{c, e}},
		{"module_id=" + core.String() + ",none", []uuid.UUID{a, b, c, e}, []uuid.UUID{b, c, e}},
		{"cycle_id=" + sprint.String() + "&module_id=" + ui.String(), []uuid.UUID{b}, []uuid.UUID{b}},
		{"cycle_id=" + sprint.String() + "&label_id=" + w.bug.String(), []uuid.UUID{a}, nil},
		{"cycle_id=" + uuid.NewV7().String() + "&module_id=none", nil, nil},
	} {
		t.Run(tc.query, func(t *testing.T) {
			listed(t, "alice", tc.query, tc.alice...)
			listed(t, "walt", tc.query, tc.walt...)
		})
	}
	t.Run("refusals", func(t *testing.T) {
		for _, tc := range []struct{ path, want string }{
			{memberIssuesPath(w, "walt") + "?relation=created&cycle_id=" + sprint.String(), "cycle_id not_allowed"},
			{memberIssuesPath(w, "walt") + "?relation=created&module_id=none", "module_id not_allowed"},
		} {
			if got := oneError(t, []byte(w.send(t, "walt", http.MethodGet, tc.path, "", http.StatusUnprocessableEntity))); got != tc.want {
				t.Errorf("%s: %s; want %s", tc.path, got, tc.want)
			}
		}
		// A repeated parameter breaks the contract's request shape, so only
		// the answer is checked against it, with and without a token.
		for _, token := range []string{w.tokens["alice"], ""} {
			for _, field := range []string{"cycle_id", "module_id"} {
				req := newRequest(t, http.MethodGet, w.base+path+"?"+field+"="+sprint.String()+"&"+field+"=none", token, nil)
				res, body := send(t, req)
				w.contract.CheckResponse(t, req, res)
				if res.StatusCode != http.StatusUnprocessableEntity || oneError(t, body) != field+" not_allowed" {
					t.Errorf("%s given twice = %d %s", field, res.StatusCode, body)
				}
			}
		}
	})
}
