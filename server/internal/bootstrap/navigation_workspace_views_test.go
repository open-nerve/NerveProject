package bootstrap

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// workspaceViewsWorld is TestWorkspaceViewsThroughTheApp's world:
// issueWorld's acme and Web, gus, acme's guest, and the work items the
// round trip filters (savedViewsRoundTrip). Web's views, cycles and
// modules are turned on, with its cycle Sprint and its module Core; Alpha
// is acme's second project, alice's, as Web is. Alice makes i1 in Web:
// urgent, Bug, assigned to bob, dated, mentioning him, in Sprint and Core;
// bob makes i2, i1's child, in Done, high; alice makes i3, plain, in Web
// and a1, urgent, in Alpha, in that order.
type workspaceViewsWorld struct {
	issueWorld
	alpha, sprint, core uuid.UUID
	i1, i2, i3, a1      uuid.UUID
}

func newWorkspaceViewsWorld(t *testing.T) workspaceViewsWorld {
	t.Helper()
	w := workspaceViewsWorld{issueWorld: newIssueWorld(t)}
	w.joinAcme(t, "gus", shared.RoleGuest)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	w.sprint = w.newCycle(t, "alice", `{"name":"Sprint"}`)
	w.core = w.newModule(t, `{"name":"Core"}`)
	w.alpha = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Alpha", "ALP")
	mention, err := json.Marshal(mentionHTML(w.ids["bob"]))
	if err != nil {
		t.Fatal(err)
	}
	w.i1 = w.newWorkItemIn(t, w.web, `{"name":"Urgent","priority":"urgent","label_ids":`+idArray(w.bug)+`,"assignee_ids":`+idArray(w.ids["bob"])+
		`,"start_date":"2026-10-12","target_date":"2026-10-15","description_html":`+string(mention)+`,"cycle_id":"`+w.sprint.String()+
		`","module_ids":`+idArray(w.core)+`}`)
	w.i2 = w.newWorkItemBy(t, "bob", w.web, `{"name":"Done child","state_id":"`+w.done.String()+`","parent_id":"`+w.i1.String()+`","priority":"high"}`)
	w.i3 = w.newWorkItemIn(t, w.web, `{"name":"Plain"}`)
	w.a1 = w.newWorkItemIn(t, w.alpha, `{"name":"Alpha urgent","priority":"urgent"}`)
	return w
}

// A workspace's own views on the real app (H14, M7 P3b spec 2.2, 2.4,
// 2.5, 4, R1), each cell a subtest in order on one world
// (newWorkspaceViewsWorld), and H13's round trip of a saved view's words to
// the lists, a workspace view's and a project view's.
func TestWorkspaceViewsThroughTheApp(t *testing.T) {
	w := newWorkspaceViewsWorld(t)
	acme := workspaceViewsPath("acme")
	var bobs, alices viewHTTP
	t.Run("create with defaults", func(t *testing.T) {
		from := time.Now().UTC().Truncate(time.Microsecond)
		v, created := w.newView(t, "bob", acme, `{"name":"  Bob's  "}`)
		want := viewHTTP{ID: v.ID, WorkspaceID: w.acme, Name: "Bob's", LogoProps: v.LogoProps, Filters: map[string][]string{},
			DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties(), OwnedByID: w.ids["bob"],
			CreatedAt: v.CreatedAt, UpdatedAt: v.CreatedAt}
		if !reflect.DeepEqual(v, want) || string(v.LogoProps) != "{}" || v.CreatedAt.Before(from) || v.CreatedAt.After(time.Now()) {
			t.Fatalf("created %s; want %+v", created, want)
		}
		if r := viewRowOf(t, w.pool, v.ID); r.createdBy != w.ids["bob"] || r.updatedBy != w.ids["bob"] || r.deletedAt != nil {
			t.Fatalf("its row: %+v", r)
		}
		if _, read := w.readView(t, "bob", v.ID); read != created {
			t.Fatalf("read %s; created %s", read, created)
		}
		bobs = v
	})
	t.Run("list", func(t *testing.T) {
		// The workspace's own views, the newest first, for its admins and
		// members alike; a project's view is its project's.
		w.newView(t, "alice", projectViewsPath(w.web), `{"name":"Web's"}`)
		alices, _ = w.newView(t, "alice", acme, `{"name":"Alice's"}`)
		for _, actor := range []string{"alice", "bob", "walt", "wanda"} {
			if got, want := w.listedViews(t, actor, acme), []uuid.UUID{alices.ID, bobs.ID}; !slices.Equal(got, want) {
				t.Errorf("%s lists %v; want %v", actor, got, want)
			}
		}
	})
	t.Run("workspace guest", func(t *testing.T) {
		// Acme's guest has no saved workspace view (R1): each operation is
		// refused, a list and a creation too.
		for _, op := range []struct{ method, path, body string }{
			{http.MethodGet, acme, ""}, {http.MethodPost, acme, `{"name":"Guest's"}`}, {http.MethodGet, viewPath(bobs.ID), ""},
			{http.MethodPatch, viewPath(bobs.ID), `{"name":"Guest's"}`}, {http.MethodDelete, viewPath(bobs.ID), ""},
		} {
			w.refusedAs(t, "gus", op.method, op.path, op.body, http.StatusForbidden, "forbidden")
		}
		if got := w.listedViews(t, "alice", acme); !slices.Equal(got, []uuid.UUID{alices.ID, bobs.ID}) {
			t.Fatal(got)
		}
	})
	t.Run("owners and admins", func(t *testing.T) {
		// Only its owner changes a view; a member deletes only his own,
		// acme's admin another's too, at one moment, the deletion his.
		w.refusedAs(t, "wanda", http.MethodPatch, viewPath(bobs.ID), `{"name":"Taken"}`, http.StatusForbidden, "forbidden")
		w.refusedAs(t, "walt", http.MethodDelete, viewPath(alices.ID), "", http.StatusForbidden, "forbidden")
		forWanda, _ := w.newView(t, "bob", acme, `{"name":"For wanda"}`)
		from := time.Now().UTC().Truncate(time.Microsecond)
		w.send(t, "wanda", http.MethodDelete, viewPath(forWanda.ID), "", http.StatusNoContent)
		viewDeletedBy(t, w.pool, forWanda.ID, w.ids["bob"], w.ids["wanda"], from, time.Now())
		if v, _ := w.changeView(t, "bob", bobs.ID, `{"name":"Bob's own"}`); v.Name != "Bob's own" || !v.UpdatedAt.After(bobs.UpdatedAt) {
			t.Fatal(v)
		}
		mine, _ := w.newView(t, "walt", acme, `{"name":"Walt's"}`)
		w.send(t, "walt", http.MethodDelete, viewPath(mine.ID), "", http.StatusNoContent)
		if got := w.listedViews(t, "bob", acme); !slices.Equal(got, []uuid.UUID{alices.ID, bobs.ID}) {
			t.Fatal(got)
		}
	})
	t.Run("refusals", func(t *testing.T) {
		// A workspace view holds the workspace list's words: no cycle or
		// module, a spreadsheet or a list, no group.
		d := shared.DefaultListDisplayFilters()
		group := "priority"
		d.Layout, d.GroupBy = "kanban", &group
		body := `{"name":"Grouped","filters":{"cycle_id":["` + w.sprint.String() + `"],"module_id":["` + w.core.String() + `"]},"display_filters":` +
			displayJSON(t, d) + `}`
		want := "validation_failed filters.cycle_id not_allowed; filters.module_id not_allowed; display_filters.layout not_allowed; display_filters.group_by not_allowed"
		w.refusedAs(t, "bob", http.MethodPost, acme, body, http.StatusUnprocessableEntity, want)
		w.refusedAs(t, "bob", http.MethodPatch, viewPath(bobs.ID), body, http.StatusUnprocessableEntity, want)
	})
	t.Run("round trip", func(t *testing.T) {
		savedViewsRoundTrip(t, w)
	})
}
