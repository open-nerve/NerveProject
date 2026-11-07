package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleLinkAnswer is a module's link as the API answers it.
type moduleLinkAnswer struct {
	ID          uuid.UUID          `json:"id"`
	ModuleID    uuid.UUID          `json:"module_id"`
	URL         string             `json:"url"`
	Title       *string            `json:"title"`
	Metadata    map[string]*string `json:"metadata"`
	CreatedByID *uuid.UUID         `json:"created_by_id"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// linkedModule is a module alice creates in Web through the API, its
// modules turned on first, named name.
func (w issueWorld) linkedModule(t *testing.T, name string) uuid.UUID {
	t.Helper()
	w.turnModulesOn(t, w.web)
	return w.createModule(t, "alice", `{"name":"`+name+`"}`, http.StatusCreated).ID
}

// addLink has the account name add a link with body to module through the
// API, checks the answer is want, and returns the link.
func (w issueWorld) addLink(t *testing.T, name string, module uuid.UUID, body string, want int) moduleLinkAnswer {
	t.Helper()
	var l moduleLinkAnswer
	decodeAnswer(t, w.send(t, name, http.MethodPost, "/api/v0/modules/"+module.String()+"/links", body, want), &l)
	return l
}

// moduleLinks are module's links as the account name lists them.
func (w issueWorld) moduleLinks(t *testing.T, name string, module uuid.UUID) []moduleLinkAnswer {
	t.Helper()
	var list struct{ Data []moduleLinkAnswer }
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/modules/"+module.String()+"/links", "", http.StatusOK), &list)
	return list.Data
}

// refusedLink has the account name send method path with body and checks
// it is refused with status and the refusal (refusalOf).
func (w issueWorld) refusedLink(t *testing.T, name, method, path, body string, status int, refusal string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, method, path, body, status)), false); got != refusal {
		t.Errorf("%s %s as %s with %s = %s, want %s", method, path, name, body, got, refusal)
	}
}

// The app adds, lists, changes and removes a module's links as the
// contract says: the address normalized, the title none or empty too, the
// metadata empty; the latest first; a title alone keeping the metadata a
// crawl wrote, a new address emptying it, a change of nothing writing
// nothing; the removal and the last change one moment, the remover's, the
// link read no more. An address the module has is refused, the link's own
// on a change is not, nor a removed link's, nor the same in another module.
func TestModuleLinksThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	billing := w.linkedModule(t, "Billing")
	docs := w.createModule(t, "alice", `{"name":"Docs"}`, http.StatusCreated).ID
	bob := w.ids["bob"]
	spec := w.addLink(t, "bob", billing, `{"url":" nerve.dev/spec ","title":""}`, http.StatusCreated)
	if spec.ModuleID != billing || spec.URL != "http://nerve.dev/spec" || spec.Title == nil || *spec.Title != "" || spec.Metadata == nil ||
		len(spec.Metadata) != 0 || spec.CreatedByID == nil || *spec.CreatedByID != bob || !spec.CreatedAt.Equal(spec.UpdatedAt) {
		t.Fatalf("Spec = %+v", spec)
	}
	plan := w.addLink(t, "alice", billing, `{"url":"https://nerve.dev/plan","title":null}`, http.StatusCreated)
	if plan.Title != nil {
		t.Fatalf("Plan's title = %q, want none", *plan.Title)
	}
	if list := w.moduleLinks(t, "bob", billing); len(list) != 2 || list[0].ID != plan.ID || list[1].ID != spec.ID {
		t.Fatalf("Billing's links = %+v, want Plan, then Spec", list)
	}
	w.refusedLink(t, "alice", http.MethodPost, "/api/v0/modules/"+billing.String()+"/links", `{"url":"http://nerve.dev/spec"}`, http.StatusConflict,
		"planning.module_link_exists")
	w.addLink(t, "alice", docs, `{"url":"http://nerve.dev/spec"}`, http.StatusCreated)

	// A crawl wrote Spec's metadata; alice, who did not add Spec, changes its title alone.
	if _, err := w.pool.Exec(pgtest.Soon(t), `UPDATE module_links SET metadata = '{"title":"Spec page"}' WHERE id = $1`, spec.ID); err != nil {
		t.Fatal(err)
	}
	path := "/api/v0/module-links/" + spec.ID.String()
	var titled moduleLinkAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPatch, path, `{"title":"Spec"}`, http.StatusOK), &titled)
	if titled.URL != spec.URL || titled.Title == nil || *titled.Title != "Spec" || titled.Metadata["title"] == nil ||
		*titled.Metadata["title"] != "Spec page" || !titled.UpdatedAt.After(spec.UpdatedAt) {
		t.Fatalf("a title alone = %+v; want the metadata kept", titled)
	}
	var same moduleLinkAnswer
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, path, `{"url":"http://nerve.dev/spec","title":"Spec"}`, http.StatusOK), &same)
	if !same.UpdatedAt.Equal(titled.UpdatedAt) {
		t.Fatalf("a change of nothing wrote: %s, was %s", same.UpdatedAt, titled.UpdatedAt)
	}
	w.refusedLink(t, "bob", http.MethodPatch, path, `{"url":"https://nerve.dev/plan"}`, http.StatusConflict, "planning.module_link_exists")
	var moved moduleLinkAnswer
	decodeAnswer(t, w.send(t, "bob", http.MethodPatch, path, `{"url":"nerve.dev/spec/v2","title":null}`, http.StatusOK), &moved)
	if moved.URL != "http://nerve.dev/spec/v2" || moved.Title != nil || len(moved.Metadata) != 0 {
		t.Fatalf("a new address = %+v; want the metadata emptied, the title cleared", moved)
	}

	w.send(t, "bob", http.MethodDelete, "/api/v0/module-links/"+plan.ID.String(), "", http.StatusNoContent)
	var stamped bool
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT deleted_at = updated_at AND updated_by_id = $2 AND deleted_at > created_at FROM module_links WHERE id = $1`,
		plan.ID, bob).Scan(&stamped); err != nil || !stamped {
		t.Fatalf("Plan's removal = %t, %v; want its last change, by bob", stamped, err)
	}
	removed := "/api/v0/module-links/" + plan.ID.String()
	w.refusedLink(t, "alice", http.MethodPatch, removed, `{"title":"A"}`, http.StatusNotFound, "planning.module_link_not_found")
	w.refusedLink(t, "alice", http.MethodDelete, removed, "", http.StatusNotFound, "planning.module_link_not_found")
	if list := w.moduleLinks(t, "alice", billing); len(list) != 1 || list[0].ID != spec.ID {
		t.Fatalf("Billing's links = %+v, want Spec alone", list)
	}
	w.addLink(t, "alice", billing, `{"url":"https://nerve.dev/plan"}`, http.StatusCreated)
}

// The refusals of the links' writes in their order (M6 P2 spec R2): values
// first (422), even on a module that is not there or by a caller who does
// not see it; then the module or the link (404), a link whose module is
// deleted too; then the role (403): a member of the workspace who is none of
// Web may not list them, and an admin of the workspace who is only Web's
// guest, who sees only his own work items there, may list them and write
// none; then the modules turned off and the module archived (409), whose
// links are listed still.
func TestModuleLinkRefusals(t *testing.T) {
	w := newIssueWorld(t)
	billing := w.linkedModule(t, "Billing")
	spec := w.addLink(t, "alice", billing, `{"url":"https://nerve.dev/spec"}`, http.StatusCreated)
	links, link := "/api/v0/modules/"+billing.String()+"/links", "/api/v0/module-links/"+spec.ID.String()
	nowhere := "/api/v0/modules/" + uuid.NewV7().String() + "/links"
	w.refusedLink(t, "alice", http.MethodPost, nowhere, `{"url":"http://u:p@nerve.dev"}`, http.StatusUnprocessableEntity, "validation_failed url not_allowed")
	w.refusedLink(t, "olga", http.MethodPatch, link, `{"url":"ftp://nerve.dev"}`, http.StatusUnprocessableEntity, "validation_failed url not_allowed")
	w.refusedLink(t, "alice", http.MethodPost, nowhere, `{"url":"nerve.dev"}`, http.StatusNotFound, "planning.module_not_found")
	w.refusedLink(t, "olga", http.MethodGet, links, "", http.StatusNotFound, "planning.module_not_found")
	w.refusedLink(t, "olga", http.MethodDelete, link, "", http.StatusNotFound, "planning.module_link_not_found")
	w.refusedLink(t, "walt", http.MethodGet, links, "", http.StatusForbidden, "forbidden")
	w.adminGuest(t)
	w.refusedLink(t, "walt", http.MethodPost, links, `{"url":"nerve.dev"}`, http.StatusForbidden, "forbidden")
	w.refusedLink(t, "walt", http.MethodPatch, link, `{"title":"A"}`, http.StatusForbidden, "forbidden")
	w.refusedLink(t, "walt", http.MethodDelete, link, "", http.StatusForbidden, "forbidden")
	if list := w.moduleLinks(t, "walt", billing); len(list) != 1 {
		t.Fatalf("walt lists %+v, want Spec", list)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	for _, write := range []struct{ method, path, body string }{{http.MethodPost, links, `{"url":"nerve.dev"}`}, {http.MethodPatch, link, `{}`},
		{http.MethodDelete, link, ""}} {
		w.refusedLink(t, "bob", write.method, write.path, write.body, http.StatusConflict, "planning.modules_disabled")
	}
	w.turnModulesOn(t, w.web)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = now() WHERE id = $1", billing); err != nil {
		t.Fatal(err)
	}
	for _, write := range []struct{ method, path, body string }{{http.MethodPost, links, `{"url":"nerve.dev"}`}, {http.MethodPatch, link, `{}`},
		{http.MethodPatch, link, `{"title":"A"}`}, {http.MethodDelete, link, ""}} {
		w.refusedLink(t, "bob", write.method, write.path, write.body, http.StatusConflict, "planning.archived")
	}
	if list := w.moduleLinks(t, "walt", billing); len(list) != 1 || list[0].Title != nil {
		t.Fatalf("walt lists %+v, want Spec as it was", list)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET archived_at = NULL, deleted_at = now() WHERE id = $1", billing); err != nil {
		t.Fatal(err)
	}
	w.refusedLink(t, "alice", http.MethodPatch, link, `{"title":"A"}`, http.StatusNotFound, "planning.module_link_not_found")
	w.refusedLink(t, "alice", http.MethodGet, links, "", http.StatusNotFound, "planning.module_not_found")
}

// addsLink is name's addition of a link with body to Web's one module.
func addsLink(name, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		var module uuid.UUID
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM modules WHERE project_id = $1", w.web).Scan(&module); err != nil {
			t.Fatal(err)
		}
		return w.sending(e, newPlanningRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/modules/"+module.String()+"/links", body)
	}
}

// Two additions of one address to a module, the second waiting on Web's
// row while the first holds it: the first is added, the second's insert
// breaks the module's exclusion of a second live address once the first has
// committed and is refused, and neither deadlocks (no 40P01, which would
// answer 500).
func TestCreateModuleLinkInterleaving(t *testing.T) {
	inBothOrdersAt(t, "projects", func(t *testing.T, w issueWorld) { w.linkedModule(t, "Billing") }, "alice",
		addsLink("alice", `{"url":"https://nerve.dev/spec"}`), "bob", addsLink("bob", `{"url":"https://nerve.dev/spec"}`),
		func(t *testing.T, w issueWorld, aliceFirst bool, alice, bob exchange) {
			first, second := alice, bob
			if !aliceFirst {
				first, second = bob, alice
			}
			w.answered(t, first, http.StatusCreated)
			w.refused(t, second, http.StatusConflict, "planning.module_link_exists")
			var n int
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM module_links WHERE project_id = $1", w.web).Scan(&n); err != nil || n != 1 {
				t.Fatalf("links of Web = %d, %v; want the first's one", n, err)
			}
		})
}
