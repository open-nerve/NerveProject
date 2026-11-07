package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// listCycles is the account name's list of Web's cycles with query, its ids in its order.
func (w issueWorld) listCycles(t *testing.T, name, query string) []uuid.UUID {
	t.Helper()
	var list struct{ Data []cycleAnswer }
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/projects/"+w.web.String()+"/cycles"+query, "", http.StatusOK), &list)
	ids := []uuid.UUID{}
	for _, c := range list.Data {
		ids = append(ids, c.ID)
	}
	return ids
}

// getCycle is the account name's read of the cycle id.
func (w issueWorld) getCycle(t *testing.T, name string, id uuid.UUID) cycleAnswer {
	t.Helper()
	var c cycleAnswer
	decodeAnswer(t, w.send(t, name, http.MethodGet, "/api/v0/cycles/"+id.String(), "", http.StatusOK), &c)
	return c
}

// newCycle is the id of the cycle the account name creates in Web with body.
func (w issueWorld) newCycle(t *testing.T, name, body string) uuid.UUID {
	t.Helper()
	var c cycleAnswer
	decodeAnswer(t, w.createCycle(t, name, body, http.StatusCreated), &c)
	return c.ID
}

// The app reads Web's cycles as the contract says: the list leaves the archived ones out, in its default order (sort
// order, then creation, then id), or, with archived=true, lists them alone, the latest archived first; each read
// answers the cycle as created, its status on Shanghai's day; a deleted cycle is in neither list and not read. Reads
// go on with the project's cycles turned off, with the project archived, and for a caller who sees only his own work
// items there; a non-member who sees Web is refused, one who does not see it, or a deleted project, is not found.
func TestReadCyclesThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	later := w.newCycle(t, "alice", datesBody("Later", shanghaiDay(t, now, 10), shanghaiDay(t, now, 20)))
	current := w.newCycle(t, "bob", datesBody("Now", shanghaiDay(t, now, -1), shanghaiDay(t, now, 1)))
	draft := w.newCycle(t, "alice", `{"name":"Draft"}`)
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	old := w.newCycle(t, "alice", datesBody("Old", shanghaiDay(t, now, -30), shanghaiDay(t, now, -20)))
	older := w.newCycle(t, "alice", datesBody("Older", shanghaiDay(t, now, -60), shanghaiDay(t, now, -50)))
	// Each is created first, so newest first; Draft moved last, Gone deleted, Old and Older archived, Old later.
	for _, change := range []struct {
		sql string
		id  uuid.UUID
	}{
		{"UPDATE cycles SET sort_order = 999999 WHERE id = $1", draft},
		{"UPDATE cycles SET deleted_at = now() WHERE id = $1", gone},
		{"UPDATE cycles SET archived_at = now() - interval '1 hour' WHERE id = $1", older},
		{"UPDATE cycles SET archived_at = now() WHERE id = $1", old},
	} {
		if _, err := w.pool.Exec(pgtest.Soon(t), change.sql, change.id); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"", "?archived=false"} {
		if got, want := w.listCycles(t, "bob", query), []uuid.UUID{current, later, draft}; !slices.Equal(got, want) {
			t.Errorf("list%s = %v, want %v", query, got, want)
		}
	}
	if got, want := w.listCycles(t, "bob", "?archived=true"), []uuid.UUID{old, older}; !slices.Equal(got, want) {
		t.Errorf("archived list = %v, want %v", got, want)
	}
	// No boolean: refused as the contract's request check would, so sent without it (M6 P1 spec R9).
	req := newRequest(t, http.MethodGet, w.base+"/api/v0/projects/"+w.web.String()+"/cycles?archived=yes", w.tokens["bob"], nil)
	res, out := send(t, req)
	w.contract.CheckResponse(t, req, res)
	if res.StatusCode != http.StatusBadRequest || refusalOf(t, out, false) != "bad_request archived invalid_format" {
		t.Errorf("archived=yes = %d %s, want 400 bad_request", res.StatusCode, out)
	}
	for id, status := range map[uuid.UUID]string{later: "upcoming", current: "current", draft: "draft", old: "completed"} {
		if c := w.getCycle(t, "bob", id); c.ID != id || c.Status != status || c.ProjectID != w.web {
			t.Errorf("GET %s = %+v, want %s", id, c, status)
		}
	}
	w.refusedRead(t, "bob", "/api/v0/cycles/"+gone.String(), http.StatusNotFound, "planning.cycle_not_found")
	w.refusedRead(t, "bob", "/api/v0/cycles/"+uuid.NewV7().String(), http.StatusNotFound, "planning.cycle_not_found")

	// Cycles turned off, then the project archived: reads go on.
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	if got := w.listCycles(t, "bob", ""); len(got) != 3 || w.getCycle(t, "bob", current).Status != "current" {
		t.Errorf("reads of an archived project with cycles off: %v", got)
	}
	// walt, acme's admin and only Web's guest, sees only his own work items there, and reads every cycle.
	w.adminGuest(t)
	if got := w.listCycles(t, "walt", ""); len(got) != 3 || w.getCycle(t, "walt", later).ID != later {
		t.Errorf("walt's reads: %v", got)
	}
	// wanda, acme's admin and none of Web's members, sees Web and may not read its cycles; olga does not see it.
	for _, tt := range []struct {
		name, path string
		status     int
		code       string
	}{
		{"wanda", "/api/v0/projects/" + w.web.String() + "/cycles", http.StatusForbidden, "forbidden"},
		{"wanda", "/api/v0/cycles/" + later.String(), http.StatusForbidden, "forbidden"},
		{"olga", "/api/v0/projects/" + w.web.String() + "/cycles", http.StatusNotFound, "project.not_found"},
		{"olga", "/api/v0/cycles/" + later.String(), http.StatusNotFound, "planning.cycle_not_found"},
	} {
		w.refusedRead(t, tt.name, tt.path, tt.status, tt.code)
	}
	// A deleted project: its cycles are gone with it at once, before the deletion job.
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", http.StatusNoContent)
	w.refusedRead(t, "bob", "/api/v0/projects/"+w.web.String()+"/cycles", http.StatusNotFound, "project.not_found")
	w.refusedRead(t, "bob", "/api/v0/cycles/"+later.String(), http.StatusNotFound, "planning.cycle_not_found")
}

// refusedRead has the account name GET path and checks it is refused with status and code.
func (w issueWorld) refusedRead(t *testing.T, name, path string, status int, code string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, http.MethodGet, path, "", status)), false); got != code {
		t.Errorf("GET %s as %s = %s, want %s", path, name, got, code)
	}
}
