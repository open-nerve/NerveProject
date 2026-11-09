package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A project's list groups its work items by cycle and by module, and
// subgroups them by either within the other (M6 P3b spec 2.1, A2): a work
// item is in its cycle's group, and in the group of each of its modules;
// one with none is in the none group, last, after the others in their keys'
// order. Memberships left never count, an archived cycle or module does,
// and one without members has no group. Each group, cell and lane counts
// its own work items, the outer total each work item once, and a group or
// a cell continues page by page in either direction. a, alice's, is in
// Sprint, Core and UI, archived; b, walt's, in Sprint and Core; c, walt's,
// in Old, archived, having left Sprint and Core; d, alice's, in UI, having
// left Gone; e, walt's, in nothing, having left Idle and UI. Walt, a member
// of Web, sees them all; as its guest he sees only his own (OwnOnly), whose
// totals count only those, and UI, which has none of his, is not a group
// of his. The member's list refuses both groups, and a project's
// preferences keep either, subgrouped by the other.
func TestIssueListPlanningGroups(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	f := listGroupFixture{w: w, groups: map[string]listGroupWant{}, keys: map[uuid.UUID]json.RawMessage{}, rows: map[uuid.UUID]listRowWant{}}
	for n, by := range []string{"alice", "walt", "walt", "alice", "walt"} {
		id := w.newWorkItemBy(t, by, w.web, fmt.Sprintf(`{"name":"Planned %d"}`, n))
		stamp := fmt.Sprintf("2030-02-01T00:00:00.%06dZ", n)
		listSeedSQL(t, w, "UPDATE issues SET created_at=$2::text::timestamptz WHERE id=$1", id, stamp)
		f.ids = append(f.ids, id)
		f.keys[id] = json.RawMessage(strconv.Quote(stamp))
		f.rows[id] = listRowWant{}
	}
	a, b, c, d, e := f.ids[0], f.ids[1], f.ids[2], f.ids[3], f.ids[4]
	sprint, old, gone := w.seedPlanningTarget(t, "cycles", "Sprint"), w.seedPlanningTarget(t, "cycles", "Old"), w.seedPlanningTarget(t, "cycles", "Gone")
	core, ui, idle := w.seedPlanningTarget(t, "modules", "Core"), w.seedPlanningTarget(t, "modules", "UI"), w.seedPlanningTarget(t, "modules", "Idle")
	listSeedSQL(t, w, "UPDATE cycles SET archived_at = now() WHERE id = $1", old)
	listSeedSQL(t, w, "UPDATE modules SET archived_at = now() WHERE id = $1", ui)
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", a, sprint, false}, {"module_issues", "module_id", a, core, false}, {"module_issues", "module_id", a, ui, false},
		{"cycle_issues", "cycle_id", b, sprint, false}, {"module_issues", "module_id", b, core, false},
		{"cycle_issues", "cycle_id", c, sprint, true}, {"cycle_issues", "cycle_id", c, old, false}, {"module_issues", "module_id", c, core, true},
		{"cycle_issues", "cycle_id", d, gone, true}, {"module_issues", "module_id", d, ui, false},
		{"module_issues", "module_id", e, idle, true}, {"module_issues", "module_id", e, ui, true},
	} {
		w.seedMembership(t, row.table, row.column, row.issue, row.target, row.gone)
	}
	// Written independently of the statements; checkListGroups keeps the
	// groups that have work items the caller sees.
	add := func(field string, members map[string][]uuid.UUID) {
		g := listGroupWant{field: field, members: members}
		for key := range members {
			if key != "none" {
				g.keys = append(g.keys, key)
			}
		}
		slices.Sort(g.keys)
		g.keys = append(g.keys, "none")
		f.groups[field] = g
	}
	add("cycle_id", map[string][]uuid.UUID{sprint.String(): {a, b}, old.String(): {c}, gone.String(): nil, "none": {d, e}})
	add("module_id", map[string][]uuid.UUID{core.String(): {a, b}, ui.String(): {a, d}, idle.String(): nil, "none": {c, e}})
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	listed := func(t *testing.T, newest ...uuid.UUID) {
		oldest := slices.Clone(newest)
		slices.Reverse(oldest)
		for _, tc := range []struct{ group, lane string }{{"cycle_id", ""}, {"module_id", ""}, {"cycle_id", "module_id"}, {"module_id", "cycle_id"}} {
			name := strings.TrimSuffix(tc.group+"/"+tc.lane, "/")
			t.Run(name+"/newest", func(t *testing.T) {
				checkListGroups(t, f, path, url.Values{}, tc.group, tc.lane, newest)
			})
			t.Run(name+"/oldest", func(t *testing.T) {
				checkListGroups(t, f, path, url.Values{"order_by": {"created_at"}}, tc.group, tc.lane, oldest)
			})
		}
	}
	t.Run("member", func(t *testing.T) { listed(t, e, d, c, b, a) })
	t.Run("own only", func(t *testing.T) {
		listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		listed(t, e, c, b)
	})
	t.Run("member list", func(t *testing.T) {
		for _, field := range []string{"cycle_id", "module_id"} {
			answer := w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "walt")+"?relation=created&group_by="+field, "", http.StatusUnprocessableEntity)
			if got := oneError(t, []byte(answer)); got != "group_by not_allowed" {
				t.Errorf("member list grouped by %s: %s", field, got)
			}
		}
	})
	t.Run("preferences", func(t *testing.T) {
		preferences := "/api/v0/me/projects/" + w.web.String() + "/preferences"
		for _, pair := range [][2]string{{"cycle_id", "module_id"}, {"module_id", "cycle_id"}} {
			display := fmt.Sprintf(`{"layout":"kanban","group_by":%q,"sub_group_by":%q,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}`, pair[0], pair[1])
			var saved struct {
				DisplayFilters json.RawMessage `json:"display_filters"`
			}
			decodeAnswer(t, w.send(t, "alice", http.MethodPatch, preferences, `{"display_filters":`+display+`}`, http.StatusOK), &saved)
			samePreferencesJSON(t, string(saved.DisplayFilters), display)
			decodeAnswer(t, w.send(t, "alice", http.MethodGet, preferences, "", http.StatusOK), &saved)
			samePreferencesJSON(t, string(saved.DisplayFilters), display)
		}
	})
}
