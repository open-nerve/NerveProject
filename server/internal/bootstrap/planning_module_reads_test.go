package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// listModules is the account name's list of Web's modules with query, its ids in its order.
func (w issueWorld) listModules(t *testing.T, name, query string) []uuid.UUID {
	t.Helper()
	var list struct{ Data []moduleAnswer }
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/projects/"+w.web.String()+"/modules"+query, "", http.StatusOK), &list)
	ids := []uuid.UUID{}
	for _, m := range list.Data {
		ids = append(ids, m.ID)
	}
	return ids
}

// getModule is the account name's read of the module id.
func (w issueWorld) getModule(t *testing.T, name string, id uuid.UUID) moduleAnswer {
	t.Helper()
	var m moduleAnswer
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/modules/"+id.String(), "", http.StatusOK), &m)
	return m
}

// The app reads Web's modules as the contract says (M6 P2 spec 2.3, R9): the list leaves the archived ones out, in its
// default order (sort order, then creation, then id), or, with archived=true, lists them alone, the latest archived
// first; each read answers the module as created, an archived one too, its members ascending without one removed; a
// deleted module is in neither list and not read. Reads go on with the project's modules turned off, with the project
// archived, for its guest and for a caller who sees only his own work items there; a non-member who sees Web is
// refused, one who does not see it, or a deleted project, is not found.
func TestReadModulesThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	alice, bob := w.ids["alice"], w.ids["bob"]
	auth := w.createModule(t, "bob", `{"name":"Auth"}`, http.StatusCreated).ID
	billing := w.createModule(t, "alice", fmt.Sprintf(`{"name":"Billing","lead_id":%q,"member_ids":[%q,%q]}`, bob, bob, alice), http.StatusCreated).ID
	docs := w.createModule(t, "alice", `{"name":"Docs"}`, http.StatusCreated).ID
	gone := w.createModule(t, "alice", `{"name":"Gone"}`, http.StatusCreated).ID
	old := w.createModule(t, "alice", `{"name":"Old","status":"completed"}`, http.StatusCreated).ID
	older := w.createModule(t, "alice", `{"name":"Older","status":"cancelled"}`, http.StatusCreated).ID
	// Each is created first, so newest first; Docs moved last, Gone deleted, Old and Older archived, Old later; bob's
	// membership of Billing removed, so alice alone remains.
	for _, change := range []struct {
		sql string
		id  uuid.UUID
	}{
		{"UPDATE modules SET sort_order = 999999 WHERE id = $1", docs},
		{"UPDATE modules SET deleted_at = now() WHERE id = $1", gone},
		{"UPDATE modules SET archived_at = now() - interval '1 hour' WHERE id = $1", older},
		{"UPDATE modules SET archived_at = now() WHERE id = $1", old},
		{"UPDATE module_members SET deleted_at = now() WHERE member_id = '" + bob.String() + "' AND module_id = $1", billing},
	} {
		if _, err := w.pool.Exec(pgtest.Soon(t), change.sql, change.id); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"", "?archived=false"} {
		if got, want := w.listModules(t, "bob", query), []uuid.UUID{billing, auth, docs}; !slices.Equal(got, want) {
			t.Errorf("list%s = %v, want %v", query, got, want)
		}
	}
	if got, want := w.listModules(t, "bob", "?archived=true"), []uuid.UUID{old, older}; !slices.Equal(got, want) {
		t.Errorf("archived list = %v, want %v", got, want)
	}
	// No boolean: refused as the contract's request check would, so sent without it (M6 P2 spec R9).
	req := newRequest(t, http.MethodGet, w.base+"/api/v0/projects/"+w.web.String()+"/modules?archived=yes", w.tokens["bob"], nil)
	res, out := send(t, req)
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusBadRequest || refusalOf(t, out, false) != "bad_request archived invalid_format" {
		t.Errorf("archived=yes = %d %s, want 400 bad_request", res.StatusCode, out)
	}
	if m := w.getModule(t, "bob", billing); m.ID != billing || m.Name != "Billing" || m.LeadID == nil || *m.LeadID != bob ||
		!slices.Equal(m.MemberIDs, []uuid.UUID{alice}) || m.ArchivedAt != nil || m.ProjectID != w.web {
		t.Errorf("GET Billing = %+v, want alice alone its member", m)
	}
	if m := w.getModule(t, "bob", old); m.ID != old || m.Status != "completed" || m.ArchivedAt == nil || m.MemberIDs == nil {
		t.Errorf("GET Old = %+v, want it archived", m)
	}
	w.refusedRead(t, "bob", "/api/v0/modules/"+gone.String(), http.StatusNotFound, "planning.module_not_found")
	w.refusedRead(t, "bob", "/api/v0/modules/"+uuid.NewV7().String(), http.StatusNotFound, "planning.module_not_found")

	// Modules turned off, then the project archived: reads go on.
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	if got := w.listModules(t, "bob", ""); len(got) != 3 || w.getModule(t, "bob", old).ID != old {
		t.Errorf("reads of an archived project with modules off: %v", got)
	}
	// walt, Web's guest, reads every module; then acme's admin as well, who sees only his own work items there, too.
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	if got := w.listModules(t, "walt", "?archived=true"); len(got) != 2 || w.getModule(t, "walt", auth).ID != auth {
		t.Errorf("the guest's reads: %v", got)
	}
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
	if got := w.listModules(t, "walt", ""); len(got) != 3 || w.getModule(t, "walt", old).ID != old {
		t.Errorf("walt's reads: %v", got)
	}
	// wanda, acme's admin and none of Web's members, sees Web and may not read its modules; olga does not see it.
	for _, tt := range []struct {
		name, path string
		status     int
		code       string
	}{
		{"wanda", "/api/v0/projects/" + w.web.String() + "/modules", http.StatusForbidden, "forbidden"},
		{"wanda", "/api/v0/modules/" + auth.String(), http.StatusForbidden, "forbidden"},
		{"olga", "/api/v0/projects/" + w.web.String() + "/modules", http.StatusNotFound, "project.not_found"},
		{"olga", "/api/v0/modules/" + auth.String(), http.StatusNotFound, "planning.module_not_found"},
	} {
		w.refusedRead(t, tt.name, tt.path, tt.status, tt.code)
	}
	// A deleted project: its modules are gone with it at once, before the deletion job.
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", http.StatusNoContent)
	w.refusedRead(t, "bob", "/api/v0/projects/"+w.web.String()+"/modules", http.StatusNotFound, "project.not_found")
	w.refusedRead(t, "bob", "/api/v0/modules/"+auth.String(), http.StatusNotFound, "planning.module_not_found")
}
