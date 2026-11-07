package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// archiveStamps are the cycle id's archive and last change.
func (w issueWorld) archiveStamps(t *testing.T, id uuid.UUID) (archived *time.Time, at time.Time, by uuid.UUID) {
	t.Helper()
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at, updated_at, updated_by_id FROM cycles WHERE id=$1", id).Scan(&archived, &at, &by); err != nil {
		t.Fatal(err)
	}
	return archived, at, by
}

// The app archives and unarchives Web's cycles as the contract says (M6 P1 spec R3): only a completed one is
// archived; bob, its member, archives alice's Old, archived and last changed by him at one moment, which leaves the
// list for the archive's; archived again, nothing is written; unarchived, it is back, and unarchived again, nothing
// is written. Turned off, archived, not seen or not allowed, the project's refusals come first.
func TestArchiveCycleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	current := w.newCycle(t, "alice", datesBody("Now", day(-1), day(1)))
	draft := w.newCycle(t, "alice", `{"name":"Draft"}`)
	old := w.newCycle(t, "alice", datesBody("Old", day(-30), day(-20)))
	path := func(id uuid.UUID, action string) string { return "/api/v0/cycles/" + id.String() + "/" + action }
	for _, id := range []uuid.UUID{current, draft} {
		w.refusedPost(t, "bob", path(id, "archive"), http.StatusConflict, "planning.cycle_not_completed")
	}
	var got cycleAnswer
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, path(old, "archive"), "", http.StatusOK), &got)
	archived, at, by := w.archiveStamps(t, old)
	if archived == nil || !archived.Equal(at) || by != w.ids["bob"] || got.ArchivedAt == nil || !got.ArchivedAt.Equal(at) || got.Status != "completed" {
		t.Fatalf("archived: %v at %s by %s; answer %+v", archived, at, by, got)
	}
	if list, archive := w.listCycles(t, "bob", ""), w.listCycles(t, "bob", "?archived=true"); slices.Contains(list, old) || !slices.Equal(archive, []uuid.UUID{old}) {
		t.Fatalf("lists after the archive: %v, %v", list, archive)
	}
	w.send(t, "alice", http.MethodPost, path(old, "archive"), "", http.StatusOK)
	if again, againAt, againBy := w.archiveStamps(t, old); !again.Equal(*archived) || !againAt.Equal(at) || againBy != by {
		t.Fatalf("archived again wrote: %v at %s by %s", again, againAt, againBy)
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, path(old, "unarchive"), "", http.StatusOK), &got)
	unarchived, at, by := w.archiveStamps(t, old)
	if unarchived != nil || by != w.ids["alice"] || got.ArchivedAt != nil || !slices.Contains(w.listCycles(t, "bob", ""), old) {
		t.Fatalf("unarchived: %v at %s by %s; answer %+v", unarchived, at, by, got)
	}
	w.send(t, "bob", http.MethodPost, path(current, "unarchive"), "", http.StatusOK)
	if _, againAt, againBy := w.archiveStamps(t, current); againBy != w.ids["alice"] || !againAt.Equal(w.getCycle(t, "bob", current).CreatedAt) {
		t.Fatalf("unarchiving a cycle not archived wrote: at %s by %s", againAt, againBy)
	}
	// The refusals before the cycle's own.
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "olga", path(old, action), http.StatusNotFound, "planning.cycle_not_found")
		w.refusedPost(t, "walt", path(old, action), http.StatusForbidden, "forbidden")
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "bob", path(old, action), http.StatusConflict, "planning.cycles_disabled")
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	for _, action := range []string{"archive", "unarchive"} {
		w.refusedPost(t, "bob", path(old, action), http.StatusConflict, "project.archived")
	}
}

// refusedPost has the account name POST path, without a body, and checks it is refused with status and code.
func (w issueWorld) refusedPost(t *testing.T, name, path string, status int, code string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, http.MethodPost, path, "", status)), false); got != code {
		t.Errorf("POST %s as %s = %s, want %s", path, name, got, code)
	}
}
