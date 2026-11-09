package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The app deletes Web's cycles as the contract says: bob, its member, may not delete alice's Sprint and deletes his
// own, deleted and last changed by him at one moment, read and listed no more, and a second deletion answers not
// found; with it the display settings of its page, alice's and his, deleted and last changed by him at that moment,
// the ones deleted before keeping their deletion and Sprint's as they were (M6 P3b spec 2.5); alice, its admin,
// deletes any, a completed and archived one too. A guest, an admin of the workspace who sees only his own work items
// there and a non-member who sees Web are forbidden; one who does not see it, not found; an archived project and
// cycles turned off refuse it.
func TestDeleteCycleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	sprint := w.newCycle(t, "alice", `{"name":"Sprint"}`)
	mine := w.newCycle(t, "bob", `{"name":"Mine"}`)
	old := w.newCycle(t, "alice", datesBody("Old", shanghaiDay(t, now, -30), shanghaiDay(t, now, -20)))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+old.String()+"/archive", "", http.StatusOK)
	path := func(id uuid.UUID) string { return "/api/v0/cycles/" + id.String() }
	mineSettings, sprintSettings := cyclePreferences(mine), cyclePreferences(sprint)
	mineSettings.seed(t, w, []string{"alice", "bob"}, []string{"bob", "walt"})
	sprintSettings.seed(t, w, []string{"alice", "bob"}, nil)
	settings, sprintRows := mineSettings.stamps(t, w), sprintSettings.rows(t, w)
	if len(settings) != 4 || len(slices.DeleteFunc(slices.Clone(settings), func(s string) bool { return s != "live" })) != 2 {
		t.Fatalf("Mine's display settings before its deletion: %q; want alice's and bob's live, two deleted", settings)
	}

	w.refusedDelete(t, "bob", path(sprint), http.StatusForbidden, "forbidden")
	w.send(t, "bob", http.MethodDelete, path(mine), "", http.StatusNoContent)
	var deleted, updated time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_at, updated_by_id FROM cycles WHERE id=$1", mine).Scan(&deleted, &updated, &by); err != nil {
		t.Fatal(err)
	}
	if !deleted.Equal(updated) || by != w.ids["bob"] {
		t.Fatalf("bob's deletion: deleted %s, changed %s by %s", deleted, updated, by)
	}
	mineSettings.wantDeletedWith(t, w, settings, deleted, by)
	if got := sprintSettings.rows(t, w); got != sprintRows {
		t.Errorf("Sprint's display settings after Mine's deletion: %s; want %s", got, sprintRows)
	}
	w.refusedRead(t, "bob", path(mine), http.StatusNotFound, "planning.cycle_not_found")
	w.refusedDelete(t, "bob", path(mine), http.StatusNotFound, "planning.cycle_not_found")
	if slices.Contains(w.listCycles(t, "bob", ""), mine) {
		t.Fatal("a deleted cycle listed")
	}
	w.send(t, "alice", http.MethodDelete, path(old), "", http.StatusNoContent)

	w.refusedDelete(t, "olga", path(sprint), http.StatusNotFound, "planning.cycle_not_found")
	w.refusedDelete(t, "walt", path(sprint), http.StatusForbidden, "forbidden")
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.refusedDelete(t, "walt", path(sprint), http.StatusForbidden, "forbidden")
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
	w.refusedDelete(t, "walt", path(sprint), http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.refusedDelete(t, "alice", path(sprint), http.StatusConflict, "planning.cycles_disabled")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedDelete(t, "alice", path(sprint), http.StatusConflict, "project.archived")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.turnCyclesOn(t, w.web)
	w.send(t, "alice", http.MethodDelete, path(sprint), "", http.StatusNoContent)
}

// refusedDelete has the account name DELETE path and checks it is refused with status and code.
func (w issueWorld) refusedDelete(t *testing.T, name, path string, status int, code string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, http.MethodDelete, path, "", status)), false); got != code {
		t.Errorf("DELETE %s as %s = %s, want %s", path, name, got, code)
	}
}
