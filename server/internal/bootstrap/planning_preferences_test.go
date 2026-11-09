package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// settingsPage is the page of a cycle or a module of Web as the tests of
// its display settings find it: its rows (pagePreferences), whose parent,
// cycles or modules, is its kind; the page's own filter key, refused, and
// the other page's, a filter of it.
type settingsPage struct {
	pagePreferences
	own, other string
}

func (p settingsPage) path() string {
	return "/api/v0/me/" + p.parent + "/" + p.target.String() + "/preferences"
}

// custom are the page's settings that the tests store: a filter of each
// kind, an empty one kept, and the other page's.
func (p settingsPage) custom() string {
	return pageSettingsJSON(`{"priority":["high"],"label_id":[],"assignee_id":["me"],"`+p.other+`":["none"]}`, customDisplay, customProperties)
}

// pageSettingsJSON are display settings of a page, as JSON.
func pageSettingsJSON(filters, display, properties string) string {
	return fmt.Sprintf(`{"filters":%s,"display_filters":%s,"display_properties":%s}`, filters, display, properties)
}

// pageDefaults are the display settings on a page of a caller who has
// stored none.
var pageDefaults = pageSettingsJSON(`{}`, defaultDisplay, defaultProperties)

// answersPageSettings is the permission matrix's check of an answer of a
// page's display settings: the settings want, the caller's own (the matrix
// seeds none).
func answersPageSettings(want string) func(t *testing.T, c caller, s seeded, answer string) {
	return func(t *testing.T, _ caller, _ seeded, answer string) { samePreferencesJSON(t, answer, want) }
}

// row is the account name's undeleted row of the page's settings, whole,
// as JSON; "" when he has none. A second row would fail the query.
func (p settingsPage) row(t *testing.T, w issueWorld, name string) string {
	t.Helper()
	var row string
	err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce((SELECT row_to_json(r)::text FROM `+p.table+` r WHERE `+p.column+` = $1 AND user_id = $2
		AND deleted_at IS NULL), '')`, p.target, w.ids[name]).Scan(&row)
	if err != nil {
		t.Fatal(err)
	}
	return row
}

// owners are the accounts of the page's undeleted rows, each by its name
// in w, in their names' order, each made and last changed by its account
// in Web and acme.
func (p settingsPage) owners(t *testing.T, w issueWorld) string {
	t.Helper()
	names := map[uuid.UUID]string{}
	for name, id := range w.ids {
		names[id] = name
	}
	var users []uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(array_agg(user_id ORDER BY user_id), '{}') FROM `+p.table+` WHERE `+p.column+` = $1
		AND deleted_at IS NULL AND created_by_id = user_id AND updated_by_id = user_id AND project_id = $2 AND workspace_id = $3`,
		p.target, w.web, w.acme).Scan(&users); err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, u := range users {
		out = append(out, names[u])
	}
	return strings.Join(out, " ")
}

// settingsPages turns Web's cycles and modules on and has alice make a
// cycle and a module that may be archived at once: the cycle Old,
// completed, and the module Done.
func settingsPages(t *testing.T, w issueWorld) []settingsPage {
	t.Helper()
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	return []settingsPage{
		{cyclePreferences(w.newCycle(t, "alice", `{"name":"Old","start_date":"2026-01-01","end_date":"2026-01-14"}`)), "cycle_id", "module_id"},
		{modulePreferences(w.newModule(t, `{"name":"Done","status":"completed"}`)), "module_id", "cycle_id"},
	}
}

func TestPlanningPreferencesAPI(t *testing.T) { exercisePlanningPreferences(t, newIssueWorld(t)) }

// As the runtime role, which the server's grants allow no more than they
// say (deploy/runtime-grants.sql).
func TestPlanningPreferencesAsRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	exercisePlanningPreferences(t, newIssueWorldOn(t, roles.serverURL, roles.server, func(*config.Config) {}))
}

// The app reads and changes each caller's own display settings on a
// cycle's and a module's page as the contract says (M6 P3b spec 2.3, R6,
// R8): the defaults while he has none, a read writing nothing; each field
// given replaced whole, the first change stored on the defaults; the
// page's own key refused by the body's shape, its words at their fields,
// the row as it was. Each caller has his own row: bob's, alice's, and
// walt's, Web's guest, then acme's admin who sees only his own work items
// in Web; wanda, who sees Web and is none of its members, is refused, and
// olga, who does not, finds no page. Removed from Web, bob is refused and
// his rows stay, read back once he is a member again. The archived cycle
// and module, their kinds turned off and the archived project change
// nothing.
func exercisePlanningPreferences(t *testing.T, w issueWorld) {
	t.Helper()
	pages := settingsPages(t, w)
	defaults := pageDefaults
	for _, p := range pages {
		samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, p.path(), "", http.StatusOK), defaults)
		if row := p.row(t, w, "bob"); row != "" {
			t.Fatalf("%s: GET wrote %s", p.parent, row)
		}
		custom := p.custom()
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), custom, http.StatusOK), custom)
		samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, p.path(), "", http.StatusOK), custom)
		// Each named top-level field replaces the old one; no deep merge.
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{"filters":{"priority":["medium"]}}`, http.StatusOK),
			pageSettingsJSON(`{"priority":["medium"]}`, customDisplay, customProperties))
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{"filters":{}}`, http.StatusOK),
			pageSettingsJSON(`{}`, customDisplay, customProperties))
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{"display_filters":`+defaultDisplay+`}`, http.StatusOK),
			pageSettingsJSON(`{}`, defaultDisplay, customProperties))
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{}`, http.StatusOK), pageSettingsJSON(`{}`, defaultDisplay, customProperties))
		// Grouped by the page's own key: accepted, one group (R8).
		own := strings.Replace(customDisplay, `"group_by":"state_id"`, `"group_by":"`+p.own+`"`, 1)
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{"display_filters":`+own+`}`, http.StatusOK),
			pageSettingsJSON(`{}`, own, customProperties))
		samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), custom, http.StatusOK), custom)
		// alice's first change: the defaults and it.
		samePreferencesJSON(t, w.send(t, "alice", http.MethodPatch, p.path(), `{"display_properties":`+customProperties+`}`, http.StatusOK),
			pageSettingsJSON(`{}`, defaultDisplay, customProperties))
		samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, p.path(), "", http.StatusOK), custom)
		checkPageWords(t, w, p)
		// walt, Web's guest, then acme's admin and Web's guest (OwnOnly).
		w.makeMember(t, "walt", w.web, shared.RoleGuest)
		samePreferencesJSON(t, w.send(t, "walt", http.MethodGet, p.path(), "", http.StatusOK), defaults)
		samePreferencesJSON(t, w.send(t, "walt", http.MethodPatch, p.path(), `{"filters":{"priority":["low"]}}`, http.StatusOK),
			pageSettingsJSON(`{"priority":["low"]}`, defaultDisplay, defaultProperties))
		membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
		w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
		samePreferencesJSON(t, w.send(t, "walt", http.MethodPatch, p.path(), `{"filters":{"priority":["urgent"]}}`, http.StatusOK),
			pageSettingsJSON(`{"priority":["urgent"]}`, defaultDisplay, defaultProperties))
		w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":15}`, http.StatusOK)
		w.send(t, "alice", http.MethodDelete, "/api/v0/project-members/"+projectMemberships(t, w.pool, w.ids["walt"], w.web)[0].String(), "",
			http.StatusNoContent)
		if got := p.owners(t, w); got != "alice bob walt" {
			t.Errorf("%s: the undeleted rows are %s's; want alice's, bob's and walt's, each his own", p.parent, got)
		}
		for _, call := range []struct{ name, method, body, code string }{
			{"wanda", http.MethodGet, "", "forbidden"}, {"wanda", http.MethodPatch, `{}`, "forbidden"},
			{"olga", http.MethodGet, "", "planning." + strings.TrimSuffix(p.parent, "s") + "_not_found"},
			{"olga", http.MethodPatch, `{}`, "planning." + strings.TrimSuffix(p.parent, "s") + "_not_found"},
		} {
			status := map[bool]int{true: http.StatusForbidden, false: http.StatusNotFound}[call.code == "forbidden"]
			if got := refusalOf(t, []byte(w.send(t, call.name, call.method, p.path(), call.body, status)), false); got != call.code {
				t.Errorf("%s %s as %s = %s; want %s", call.method, p.parent, call.name, got, call.code)
			}
		}
	}
	removedAndBack(t, w, pages)
	// The cycle and the module archived, then their kinds turned off, then
	// Web archived: each changes nothing.
	for _, change := range []struct {
		priority string
		make     func()
	}{
		{"urgent", func() {
			for _, p := range pages {
				w.send(t, "alice", http.MethodPost, "/api/v0/"+p.parent+"/"+p.target.String()+"/archive", "", http.StatusOK)
			}
		}},
		{"medium", func() {
			w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false,"module_view":false}`, http.StatusOK)
		}},
		{"low", func() {
			w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
		}},
	} {
		change.make()
		for _, p := range pages {
			samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, p.path(), "", http.StatusOK), p.custom())
			changed := pageSettingsJSON(`{"priority":["`+change.priority+`"]}`, customDisplay, customProperties)
			samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), `{"filters":{"priority":["`+change.priority+`"]}}`, http.StatusOK), changed)
			samePreferencesJSON(t, w.send(t, "bob", http.MethodPatch, p.path(), p.custom(), http.StatusOK), p.custom())
		}
	}
}

// removedAndBack: bob, removed from Web, is refused on each page, his rows
// as they were; a member again, he reads them back, unchanged.
func removedAndBack(t *testing.T, w issueWorld, pages []settingsPage) {
	t.Helper()
	rows := map[string]string{}
	for _, p := range pages {
		rows[p.parent] = p.row(t, w, "bob")
	}
	w.send(t, "alice", http.MethodDelete, bobsMembership(t, w), "", http.StatusNoContent)
	for _, p := range pages {
		for _, method := range []string{http.MethodGet, http.MethodPatch} {
			body := map[string]string{http.MethodGet: "", http.MethodPatch: `{"filters":{}}`}[method]
			if got := refusalOf(t, []byte(w.send(t, "bob", method, p.path(), body, http.StatusForbidden)), false); got != "forbidden" {
				t.Errorf("%s %s as bob removed = %s; want forbidden", method, p.parent, got)
			}
		}
		if got := p.row(t, w, "bob"); got != rows[p.parent] {
			t.Errorf("%s: bob's row after his removal %s; want %s", p.parent, got, rows[p.parent])
		}
	}
	w.makeMember(t, "bob", w.web, shared.RoleMember)
	for _, p := range pages {
		samePreferencesJSON(t, w.send(t, "bob", http.MethodGet, p.path(), "", http.StatusOK), p.custom())
		if got := p.row(t, w, "bob"); got != rows[p.parent] {
			t.Errorf("%s: bob's row once he is back %s; want %s", p.parent, got, rows[p.parent])
		}
	}
}

// checkPageWords: the page's own key, at the root of its filters, refused
// by the body's shape (R8); words no list has, each at its field, in
// order; neither changes bob's row.
func checkPageWords(t *testing.T, w issueWorld, p settingsPage) {
	t.Helper()
	before := p.row(t, w, "bob")
	for _, body := range []string{`{"filters":{"` + p.own + `":[]}}`, `{"filters":{"` + p.own + `":["none"],"priority":["low"]}}`} {
		answer := draftRequest(t, w, "bob", http.MethodPatch, p.path(), body, http.StatusBadRequest)
		if got, want := refusalOf(t, []byte(answer), false), "bad_request filters."+p.own+" not_allowed"; got != want {
			t.Errorf("%s: PATCH %s = %s; want %s", p.parent, body, got, want)
		}
	}
	body := `{"filters":{"` + p.other + `":["bad-uuid","none"],"label_id":["none","me"],"target_date":["none","none"]},"display_filters":` +
		`{"layout":"future","group_by":"project_id","sub_group_by":"project_id","order_by":"-sort_order","sub_issue":false,"show_empty_groups":false,` +
		`"calendar":{"layout":"year","show_weekends":false}}}`
	// The filters' keys in their order: cycle_id before label_id, module_id after it.
	keys := []string{"filters." + p.other + "[0] not_allowed", "filters.label_id[1] not_allowed"}
	if p.other > "label_id" {
		keys[0], keys[1] = keys[1], keys[0]
	}
	refusedWords(t, w.send(t, "bob", http.MethodPatch, p.path(), body, http.StatusUnprocessableEntity), append(keys, "filters.target_date[0] not_allowed",
		"filters.target_date[1] not_allowed", "display_filters.layout not_allowed", "display_filters.group_by not_allowed",
		"display_filters.sub_group_by not_allowed", "display_filters.order_by not_allowed", "display_filters.calendar.layout not_allowed")...)
	if after := p.row(t, w, "bob"); after != before {
		t.Errorf("%s: refused words changed bob's row: %s; want %s", p.parent, after, before)
	}
}
