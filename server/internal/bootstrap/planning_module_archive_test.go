package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleStamps are the module id's archive and last change.
func (w issueWorld) moduleStamps(t *testing.T, id uuid.UUID) (archived *time.Time, at time.Time, by uuid.UUID) {
	t.Helper()
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at, updated_at, updated_by_id FROM modules WHERE id=$1", id).Scan(&archived, &at, &by); err != nil {
		t.Fatal(err)
	}
	return archived, at, by
}

// The app archives and unarchives Web's modules as the contract says (M6 P2 spec R3): only a completed or a
// cancelled one is archived; bob, its member, archives alice's Done, archived and last changed by him at one moment,
// which leaves the list for the archive's, and her Dropped; archived again, nothing is written; unarchived, whatever
// its status, it is back, and unarchived again, nothing is written. Not seen, not allowed, turned off or archived,
// the project's refusals come first.
func TestArchiveModuleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	auth := w.createModule(t, "alice", `{"name":"Auth","status":"in-progress"}`, http.StatusCreated).ID
	done := w.createModule(t, "alice", `{"name":"Done","status":"completed"}`, http.StatusCreated).ID
	dropped := w.createModule(t, "alice", `{"name":"Dropped","status":"cancelled"}`, http.StatusCreated).ID
	path := func(id uuid.UUID, action string) string { return "/api/v0/modules/" + id.String() + "/" + action }
	w.refusedPost(t, "bob", path(auth, "archive"), http.StatusConflict, "planning.module_not_closed")
	if _, at, by := w.moduleStamps(t, auth); by != w.ids["alice"] || !at.Equal(w.getModule(t, "bob", auth).CreatedAt) {
		t.Fatalf("a refused archive wrote: at %s by %s", at, by)
	}
	var got moduleAnswer
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, path(done, "archive"), "", http.StatusOK), &got)
	archived, at, by := w.moduleStamps(t, done)
	if archived == nil || !archived.Equal(at) || by != w.ids["bob"] || got.ArchivedAt == nil || !got.ArchivedAt.Equal(at) || got.Status != "completed" ||
		got.UpdatedByID == nil || *got.UpdatedByID != by {
		t.Fatalf("archived: %v at %s by %s; answer %+v", archived, at, by, got)
	}
	w.send(t, "bob", http.MethodPost, path(dropped, "archive"), "", http.StatusOK)
	if list, archive := w.listModules(t, "bob", ""), w.listModules(t, "bob", "?archived=true"); !slices.Equal(list, []uuid.UUID{auth}) ||
		len(archive) != 2 || !slices.Contains(archive, done) || !slices.Contains(archive, dropped) {
		t.Fatalf("lists after the archives: %v, %v", list, archive)
	}
	w.send(t, "alice", http.MethodPost, path(done, "archive"), "", http.StatusOK)
	if again, againAt, againBy := w.moduleStamps(t, done); !again.Equal(*archived) || !againAt.Equal(at) || againBy != by {
		t.Fatalf("archived again wrote: %v at %s by %s", again, againAt, againBy)
	}
	// Unarchived whatever its status: Dropped, planned since it was archived.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE modules SET status = 'planned' WHERE id = $1", dropped); err != nil {
		t.Fatal(err)
	}
	for _, id := range []uuid.UUID{done, dropped} {
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, path(id, "unarchive"), "", http.StatusOK), &got)
		unarchived, at, by := w.moduleStamps(t, id)
		if unarchived != nil || by != w.ids["alice"] || !at.Equal(got.UpdatedAt) || got.ArchivedAt != nil || !slices.Contains(w.listModules(t, "bob", ""), id) {
			t.Fatalf("unarchived: %v at %s by %s; answer %+v", unarchived, at, by, got)
		}
	}
	w.send(t, "bob", http.MethodPost, path(auth, "unarchive"), "", http.StatusOK)
	if _, againAt, againBy := w.moduleStamps(t, auth); againBy != w.ids["alice"] || !againAt.Equal(w.getModule(t, "bob", auth).CreatedAt) {
		t.Fatalf("unarchiving a module not archived wrote: at %s by %s", againAt, againBy)
	}
	// The refusals before the module's own.
	w.adminGuest(t)
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "olga", path(done, action), http.StatusNotFound, "planning.module_not_found")
		w.refusedPost(t, "walt", path(done, action), http.StatusForbidden, "forbidden")
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "bob", path(done, action), http.StatusConflict, "planning.modules_disabled")
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "bob", path(done, action), http.StatusConflict, "project.archived")
	}
}

// moduleRowStamps are the rows of table that hang from the module id, or the module's own row for modules, each as
// its deletion and last change; a row live as "live".
func (w issueWorld) moduleRowStamps(t *testing.T, table string, id uuid.UUID) []string {
	t.Helper()
	column := "module_id"
	if table == "modules" {
		column = "id"
	}
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT CASE WHEN deleted_at IS NULL THEN 'live'
		ELSE concat_ws(' ', deleted_at = updated_at, updated_at, updated_by_id) END FROM `+table+` WHERE `+column+` = $1 ORDER BY id`, id)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}

// The app deletes Web's modules as the contract says (M6 P2 spec 2.4, R15): bob, its member, may not delete alice's
// Auth and deletes his own Mine, archived, with its members and links: the module, each of its members' rows and
// each of its links soft-deleted at one moment by him, a link removed before keeping its removal; it is read and
// listed no more, nor are its links, which are not found to change or remove, and a second deletion answers not
// found. alice, its admin, deletes any. A guest, an admin of the workspace who sees only his own work items there and
// a non-member who sees Web are forbidden; one who does not see it, not found; modules turned off and an archived
// project refuse it.
func TestDeleteModuleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnModulesOn(t, w.web)
	alice, bob := w.ids["alice"], w.ids["bob"]
	auth := w.createModule(t, "alice", `{"name":"Auth"}`, http.StatusCreated).ID
	mine := w.createModule(t, "bob", fmt.Sprintf(`{"name":"Mine","status":"completed","member_ids":[%q,%q]}`, alice, bob), http.StatusCreated).ID
	spec := w.addLink(t, "bob", mine, `{"url":"http://169.254.169.254/spec"}`, http.StatusCreated)
	plan := w.addLink(t, "alice", mine, `{"url":"http://169.254.169.254/plan"}`, http.StatusCreated)
	gone := w.addLink(t, "alice", mine, `{"url":"http://169.254.169.254/gone"}`, http.StatusCreated)
	w.send(t, "alice", http.MethodDelete, "/api/v0/module-links/"+gone.ID.String(), "", http.StatusNoContent)
	removed := w.moduleRowStamps(t, "module_links", mine)
	w.send(t, "bob", http.MethodPost, "/api/v0/modules/"+mine.String()+"/archive", "", http.StatusOK)
	path := func(id uuid.UUID) string { return "/api/v0/modules/" + id.String() }

	w.refusedDelete(t, "bob", path(auth), http.StatusForbidden, "forbidden")
	w.send(t, "bob", http.MethodDelete, path(mine), "", http.StatusNoContent)
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM modules WHERE id = $1", mine).Scan(&at); err != nil {
		t.Fatal(err)
	}
	var stamp string
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT concat_ws(' ', true, $1::timestamptz, $2::uuid)", at, bob).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	wantLinks := slices.Clone(removed)
	for i, s := range wantLinks {
		if s == "live" {
			wantLinks[i] = stamp
		}
	}
	for table, want := range map[string][]string{"modules": {stamp}, "module_members": {stamp, stamp}, "module_links": wantLinks} {
		if got := w.moduleRowStamps(t, table, mine); !slices.Equal(got, want) {
			t.Errorf("Mine's %s after bob's deletion: %q; want %q", table, got, want)
		}
	}
	if live := slices.DeleteFunc(slices.Clone(removed), func(s string) bool { return s != "live" }); len(live) != 2 || len(removed) != 3 {
		t.Fatalf("Mine's links before its deletion: %q; want Spec and Plan live, Gone removed", removed)
	}
	w.refusedRead(t, "bob", path(mine), http.StatusNotFound, "planning.module_not_found")
	w.refusedRead(t, "bob", path(mine)+"/links", http.StatusNotFound, "planning.module_not_found")
	for _, link := range []uuid.UUID{spec.ID, plan.ID} {
		w.refusedLink(t, "alice", http.MethodPatch, "/api/v0/module-links/"+link.String(), `{"title":"A"}`, http.StatusNotFound, "planning.module_link_not_found")
		w.refusedLink(t, "alice", http.MethodDelete, "/api/v0/module-links/"+link.String(), "", http.StatusNotFound, "planning.module_link_not_found")
	}
	w.refusedDelete(t, "bob", path(mine), http.StatusNotFound, "planning.module_not_found")
	for _, query := range []string{"", "?archived=true"} {
		if slices.Contains(w.listModules(t, "bob", query), mine) {
			t.Fatalf("a deleted module listed (%q)", query)
		}
	}

	w.refusedDelete(t, "olga", path(auth), http.StatusNotFound, "planning.module_not_found")
	w.refusedDelete(t, "walt", path(auth), http.StatusForbidden, "forbidden")
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.refusedDelete(t, "walt", path(auth), http.StatusForbidden, "forbidden")
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
	w.refusedDelete(t, "walt", path(auth), http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"module_view":false}`, http.StatusOK)
	w.refusedDelete(t, "alice", path(auth), http.StatusConflict, "planning.modules_disabled")
	w.turnModulesOn(t, w.web)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedDelete(t, "alice", path(auth), http.StatusConflict, "project.archived")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, path(auth), "", http.StatusNoContent)
}
