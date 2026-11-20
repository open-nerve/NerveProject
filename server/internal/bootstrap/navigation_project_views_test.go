package bootstrap

import (
	"bytes"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newProjectViewsWorld is TestProjectViewsThroughTheApp's world:
// issueWorld's acme and Web, in an app whose deletion jobs outlast the
// test's row locks (barriersOutlasted), with Web's views turned on by
// alice, its admin. Besides bob, Web's member: max is Web's member who is
// acme's admin (Grant.ProjectAdmin by his workspace role); walt, acme's
// member, is Web's guest, who sees only his own (Grant.OwnOnly); adam is
// Web's guest who is acme's admin (adminGuest's case: Grant.ProjectAdmin
// and OwnOnly). Each joins Web as acme's member before he is made its
// admin: a workspace admin joins a project as its admin. olga is no member
// of acme.
func newProjectViewsWorld(t *testing.T) issueWorld {
	t.Helper()
	w := newIssueWorldWith(t, barriersOutlasted)
	for _, m := range []struct {
		name string
		role shared.Role
	}{{"max", shared.RoleMember}, {"adam", shared.RoleGuest}} {
		w.joinAcme(t, m.name, shared.RoleMember)
		w.makeMember(t, m.name, w.web, m.role)
		w.promoteInAcme(t, m.name)
	}
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.changeProject(t, w.web, `{"issue_views_view":true}`)
	return w
}

// A project's views on the real app (H12, M7 P3b spec 2.2, 2.4, 2.5, 4,
// R2, R6), each cell a subtest in order on one world
// (newProjectViewsWorld): bob's first view, made in the first, is the one
// the others name as another's.
func TestProjectViewsThroughTheApp(t *testing.T) {
	w := newProjectViewsWorld(t)
	web := projectViewsPath(w.web)
	var bobs viewHTTP
	t.Run("create with defaults", func(t *testing.T) {
		from := time.Now().UTC().Truncate(time.Microsecond)
		v, created := w.newView(t, "bob", web, `{"name":"  Bob's  "}`)
		// The name without its white space; no description, icon or
		// filter; the shared display settings; bob its owner, at once.
		want := viewHTTP{ID: v.ID, WorkspaceID: w.acme, ProjectID: &w.web, Name: "Bob's", LogoProps: v.LogoProps, Filters: map[string][]string{},
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
	t.Run("owners and admins", func(t *testing.T) {
		alices, _ := w.newView(t, "alice", web, `{"name":"Alice's"}`)
		forMax, _ := w.newView(t, "bob", web, `{"name":"For max"}`)
		forAlice, _ := w.newView(t, "bob", web, `{"name":"For alice"}`)
		// Only its owner changes a view (D29), whoever administers it.
		for _, admin := range []string{"alice", "max"} {
			w.refusedAs(t, admin, http.MethodPatch, viewPath(bobs.ID), `{"name":"Taken"}`, http.StatusForbidden, "forbidden")
		}
		// A member deletes only his own.
		w.refusedAs(t, "bob", http.MethodDelete, viewPath(alices.ID), "", http.StatusForbidden, "forbidden")
		// Its project's admins delete another's: Web's admin, and acme's
		// admin who is Web's member. The deletion is theirs, at one moment.
		for _, c := range []struct {
			admin string
			view  uuid.UUID
		}{{"max", forMax.ID}, {"alice", forAlice.ID}} {
			from := time.Now().UTC().Truncate(time.Microsecond)
			w.send(t, c.admin, http.MethodDelete, viewPath(c.view), "", http.StatusNoContent)
			viewDeletedBy(t, w.pool, c.view, w.ids["bob"], w.ids[c.admin], from, time.Now())
			w.refusedAs(t, "bob", http.MethodGet, viewPath(c.view), "", http.StatusNotFound, "navigation.view_not_found")
		}
		if v, _ := w.changeView(t, "bob", bobs.ID, `{"name":"Bob's own"}`); v.Name != "Bob's own" {
			t.Fatal(v)
		}
		bobs, _ = w.readView(t, "bob", bobs.ID)
	})
	t.Run("own only", func(t *testing.T) {
		ownOnly(t, w, bobs)
	})
	t.Run("views turned off", func(t *testing.T) {
		// Writes are refused, reads go on (M7 design 3.7).
		w.changeProject(t, w.web, `{"issue_views_view":false}`)
		w.refusedAs(t, "bob", http.MethodPost, web, `{"name":"Off"}`, http.StatusConflict, "navigation.views_disabled")
		w.refusedAs(t, "bob", http.MethodPatch, viewPath(bobs.ID), `{"name":"Off"}`, http.StatusConflict, "navigation.views_disabled")
		w.refusedAs(t, "bob", http.MethodDelete, viewPath(bobs.ID), "", http.StatusConflict, "navigation.views_disabled")
		if !slices.Contains(w.listedViews(t, "bob", web), bobs.ID) {
			t.Fatal("bob's view is not listed")
		}
		if v, _ := w.readView(t, "bob", bobs.ID); !reflect.DeepEqual(v, bobs) {
			t.Fatalf("read %+v; want %+v", v, bobs)
		}
		w.changeProject(t, w.web, `{"issue_views_view":true}`)
	})
	t.Run("archived project", func(t *testing.T) {
		// Views are read and written in an archived project as in any (R2).
		shelf := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shelf", "SHF")
		w.makeMember(t, "bob", shelf, shared.RoleMember)
		w.changeProject(t, shelf, `{"issue_views_view":true}`)
		kept, _ := w.newView(t, "bob", projectViewsPath(shelf), `{"name":"Kept"}`)
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+shelf.String()+"/archive", "", http.StatusOK)
		made, _ := w.newView(t, "bob", projectViewsPath(shelf), `{"name":"Made"}`)
		if got := w.listedViews(t, "bob", projectViewsPath(shelf)); !slices.Equal(got, []uuid.UUID{made.ID, kept.ID}) {
			t.Fatal(got)
		}
		w.readView(t, "bob", kept.ID)
		w.changeView(t, "bob", kept.ID, `{"name":"Changed"}`)
		w.send(t, "bob", http.MethodDelete, viewPath(made.ID), "", http.StatusNoContent)
	})
	t.Run("patch", func(t *testing.T) {
		patchesWholeFields(t, w)
	})
	t.Run("refusals", func(t *testing.T) {
		// One 422: the name's, the description's and the icon's problems,
		// then the words', the filters' by key, then the display's.
		d := shared.DefaultListDisplayFilters()
		d.Layout, d.OrderBy = "gallery", "name"
		bad := `{"name":"  ","description":"a\u0000b","logo_props":{"in_use":"emoji","emoji":{"value":"x\u0000"}},` +
			`"filters":{"priority":["now"],"cycle_id":["soon"]},"display_filters":` + displayJSON(t, d) + `}`
		want := "validation_failed name too_short; description invalid_format; logo_props.emoji.value invalid_format; " +
			"filters.cycle_id[0] not_allowed; filters.priority[0] not_allowed; display_filters.layout not_allowed; display_filters.order_by not_allowed"
		before := w.listedViews(t, "alice", web)
		w.refusedAs(t, "bob", http.MethodPost, web, bad, http.StatusUnprocessableEntity, want)
		w.refusedAs(t, "bob", http.MethodPatch, viewPath(bobs.ID), bad, http.StatusUnprocessableEntity, want)
		// A workspace's view's key is no project view's.
		w.refusedAs(t, "bob", http.MethodPost, web, `{"name":"Across","filters":{"project_id":["`+w.web.String()+`"]}}`,
			http.StatusUnprocessableEntity, "validation_failed filters.project_id not_allowed")
		if after := w.listedViews(t, "alice", web); !slices.Equal(after, before) {
			t.Fatalf("listed %v; before %v", after, before)
		}
		if v, _ := w.readView(t, "bob", bobs.ID); !reflect.DeepEqual(v, bobs) {
			t.Fatalf("read %+v; want %+v", v, bobs)
		}
	})
	t.Run("list order", func(t *testing.T) {
		listsTheNewestFirst(t, w)
	})
	t.Run("view not found", func(t *testing.T) {
		viewsNotFoundAlike(t, w, bobs)
	})
	t.Run("deletion window", func(t *testing.T) {
		viewsDeletionWindow(t, w)
	})
}

// ownOnly is the guest rule's cell (P3b spec 2.2, DP-15): in Web, which
// does not show its guests everything, walt, its guest, and adam, its
// guest who is acme's admin, each list, read, change and delete as theirs
// only their own, and every other view, bob's, answers them as an id that
// never was; a guest's role writes no view, not even his own. Web showing
// them everything, they see every view, and adam, Web's admin as acme's,
// deletes another's but changes only his own.
func ownOnly(t *testing.T, w issueWorld, bobs viewHTTP) {
	t.Helper()
	web := projectViewsPath(w.web)
	walts := seedView(t, w.pool, w.acme, &w.web, "Walt's", w.ids["walt"], time.Now().UTC().Truncate(time.Microsecond))
	adams, _ := w.newView(t, "adam", web, `{"name":"Adam's"}`)
	absent := viewPath(uuid.NewV7())
	for actor, own := range map[string]uuid.UUID{"walt": walts, "adam": adams.ID} {
		if got := w.listedViews(t, actor, web); !slices.Equal(got, []uuid.UUID{own}) {
			t.Errorf("%s lists %v; want his own %s", actor, got, own)
		}
		w.readView(t, actor, own)
		for _, op := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPatch, `{"name":"Taken"}`}, {http.MethodDelete, ""}} {
			w.answeredAsAbsent(t, actor, op.method, viewPath(bobs.ID), absent, op.body, "navigation.view_not_found")
		}
	}
	if v, _ := w.readView(t, "bob", bobs.ID); !reflect.DeepEqual(v, bobs) {
		t.Fatalf("bob's view %+v; want %+v", v, bobs)
	}
	w.refusedAs(t, "walt", http.MethodPatch, viewPath(walts), `{"name":"Walt's own"}`, http.StatusForbidden, "forbidden")
	w.refusedAs(t, "walt", http.MethodDelete, viewPath(walts), "", http.StatusForbidden, "forbidden")
	w.changeView(t, "adam", adams.ID, `{"name":"Adam's own"}`)

	w.changeProject(t, w.web, `{"guest_view_all_features":true}`)
	every := w.listedViews(t, "alice", web)
	for _, actor := range []string{"walt", "adam"} {
		if got := w.listedViews(t, actor, web); !slices.Equal(got, every) {
			t.Errorf("%s lists %v; want %v", actor, got, every)
		}
	}
	w.readView(t, "walt", bobs.ID)
	w.refusedAs(t, "walt", http.MethodDelete, viewPath(bobs.ID), "", http.StatusForbidden, "forbidden")
	w.refusedAs(t, "adam", http.MethodPatch, viewPath(bobs.ID), `{"name":"Taken"}`, http.StatusForbidden, "forbidden")
	forAdam, _ := w.newView(t, "bob", web, `{"name":"For adam"}`)
	from := time.Now().UTC().Truncate(time.Microsecond)
	w.send(t, "adam", http.MethodDelete, viewPath(forAdam.ID), "", http.StatusNoContent)
	viewDeletedBy(t, w.pool, forAdam.ID, w.ids["bob"], w.ids["adam"], from, time.Now())
	w.changeProject(t, w.web, `{"guest_view_all_features":false}`)
}

// patchesWholeFields is PATCH's cell (P3b spec 2.4): each field given
// replaces that field whole, an empty filter kept apart from one left
// out, each field left out is kept; a change of nothing, {} or the values
// the view has, after the name's white space is dropped, writes nothing:
// the answer is the same bytes as the view read before, its row's last
// change kept.
func patchesWholeFields(t *testing.T, w issueWorld) {
	t.Helper()
	v, _ := w.newView(t, "bob", projectViewsPath(w.web), `{"name":"Patched","description":"Urgent ones","filters":{"priority":["urgent"],"state_group":["started"]}}`)
	got, _ := w.changeView(t, "bob", v.ID, `{"filters":{"priority":["high"],"label_id":[]}}`)
	if !reflect.DeepEqual(got.Filters, map[string][]string{"priority": {"high"}, "label_id": {}}) || got.Name != v.Name || got.Description != v.Description ||
		!got.CreatedAt.Equal(v.CreatedAt) || !got.UpdatedAt.After(v.UpdatedAt) || !reflect.DeepEqual(got.DisplayFilters, v.DisplayFilters) {
		t.Fatalf("changed %+v; from %+v", got, v)
	}
	d := shared.DefaultListDisplayFilters()
	group := "state_group"
	d.Layout, d.GroupBy, d.OrderBy = "kanban", &group, "priority"
	if got, _ = w.changeView(t, "bob", v.ID, `{"display_filters":`+displayJSON(t, d)+`}`); !reflect.DeepEqual(got.DisplayFilters, d) {
		t.Fatalf("display %+v; want %+v", got.DisplayFilters, d)
	}
	_, before := w.readView(t, "bob", v.ID)
	row := viewRowOf(t, w.pool, v.ID)
	for _, body := range []string{`{}`, `{"name":"  Patched "}`, `{"filters":{"label_id":[],"priority":["high"]},"display_filters":` + displayJSON(t, d) + `}`,
		`{"description":"Urgent ones","logo_props":{}}`} {
		if _, after := w.changeView(t, "bob", v.ID, body); after != before {
			t.Errorf("PATCH %s: %s; want %s", body, after, before)
		}
	}
	if r := viewRowOf(t, w.pool, v.ID); !r.updatedAt.Equal(row.updatedAt) || r.updatedBy != row.updatedBy {
		t.Fatalf("a change of nothing wrote %+v; was %+v", r, row)
	}
}

// listsTheNewestFirst is the list order's cell (P3b spec 2.4): newest
// first by created_at, then by id, both descending, in Order, a project
// of three views: two made at one moment, seeded, the one with the smaller
// id written first, and a newer one alice makes. The table is analyzed
// first, so the planner reads it as it is, one page, scans it and sorts:
// the partial index's own order, (created_at DESC, id DESC), would put
// the two in order without the query's tie-break (P3b T6), and the sort
// alone keeps them as written.
func listsTheNewestFirst(t *testing.T, w issueWorld) {
	t.Helper()
	order := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Order", "ORD")
	w.changeProject(t, order, `{"issue_views_view":true}`)
	at := time.Now().UTC().Truncate(time.Microsecond).Add(-time.Hour)
	low := seedView(t, w.pool, w.acme, &order, "Earlier id", w.ids["alice"], at)
	high := seedView(t, w.pool, w.acme, &order, "Later id", w.ids["alice"], at)
	if bytes.Compare(low[:], high[:]) >= 0 {
		t.Fatalf("seeded %s before %s", low, high)
	}
	newest, _ := w.newView(t, "alice", projectViewsPath(order), `{"name":"Newest"}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "ANALYZE issue_views"); err != nil {
		t.Fatal(err)
	}
	if got, want := w.listedViews(t, "alice", projectViewsPath(order)), []uuid.UUID{newest.ID, high, low}; !slices.Equal(got, want) {
		t.Fatalf("listed %v; want %v", got, want)
	}
}

// viewsNotFoundAlike is /views/{view_id}'s 404 (P3b spec 2.5): a view in
// Vault, a private project walt is no member of, a deleted view, and
// bob's, which walt, Web's guest who sees only his own, does not see,
// each answer him a read, a change and a deletion as an id that never was,
// byte for byte; and so they answer olga, who sees no project of acme.
func viewsNotFoundAlike(t *testing.T, w issueWorld, bobs viewHTTP) {
	t.Helper()
	vault := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Vault","identifier":"VLT","network":0}`)
	w.changeProject(t, vault, `{"issue_views_view":true}`)
	sealed, _ := w.newView(t, "alice", projectViewsPath(vault), `{"name":"Sealed"}`)
	gone, _ := w.newView(t, "bob", projectViewsPath(w.web), `{"name":"Gone"}`)
	w.send(t, "bob", http.MethodDelete, viewPath(gone.ID), "", http.StatusNoContent)
	absent := viewPath(uuid.NewV7())
	for _, actor := range []string{"walt", "olga"} {
		for _, id := range []uuid.UUID{sealed.ID, gone.ID, bobs.ID} {
			for _, op := range []struct{ method, body string }{{http.MethodGet, ""}, {http.MethodPatch, `{"name":"Taken"}`}, {http.MethodDelete, ""}} {
				w.answeredAsAbsent(t, actor, op.method, viewPath(id), absent, op.body, "navigation.view_not_found")
			}
		}
	}
}
