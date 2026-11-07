package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// changeCycle has the account name PATCH the cycle id with body, checks the answer is want, and returns its body.
func (w issueWorld) changeCycle(t *testing.T, name string, id uuid.UUID, body string, want int) string {
	t.Helper()
	return w.send(t, name, http.MethodPatch, "/api/v0/cycles/"+id.String(), body, want)
}

// refusedChange has the account name PATCH the cycle id with body and checks it is refused with status and the
// refusal (refusalOf).
func (w issueWorld) refusedChange(t *testing.T, name string, id uuid.UUID, body string, status int, refusal string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.changeCycle(t, name, id, body, status)), false); got != refusal {
		t.Errorf("PATCH %s as %s with %s = %s, want %s", id, name, body, got, refusal)
	}
}

// cycleStamps are the cycle id's last change: when, and by whom.
func (w issueWorld) cycleStamps(t *testing.T, id uuid.UUID) (time.Time, uuid.UUID) {
	t.Helper()
	var at time.Time
	var by uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at, updated_by_id FROM cycles WHERE id=$1", id).Scan(&at, &by); err != nil {
		t.Fatal(err)
	}
	return at, by
}

// The app changes Web's cycles as the contract says (M6 P1 spec R2, R4, R5): bob, its member, changes alice's
// Sprint, its owner kept and the change his; a change of nothing writes nothing; the dates given alone are held to
// the cycle's, refused at the date given; a cycle may move onto its own days and not onto another's; a completed
// cycle takes its sort order alone, any other field refused even as it is, after a value's problem; an archived one
// takes no change at all, none too.
func TestUpdateCycleThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-1), day(1)))
	later := w.newCycle(t, "alice", datesBody("Later", day(10), day(20)))
	old := w.newCycle(t, "alice", datesBody("Old", day(-30), day(-20)))
	draft := w.newCycle(t, "alice", `{"name":"Draft"}`)

	var got cycleAnswer
	decodeAnswer(t, w.changeCycle(t, "bob", sprint, fmt.Sprintf(`{"name":" Sprint 1 ","description":"Ship","end_date":%q}`, day(2)), http.StatusOK), &got)
	if got.Name != "Sprint 1" || got.Description != "Ship" || *got.StartDate != day(-1) || *got.EndDate != day(2) || got.Status != "current" ||
		*got.OwnedByID != w.ids["alice"] || *got.CreatedByID != w.ids["alice"] || *got.UpdatedByID != w.ids["bob"] || !got.UpdatedAt.After(got.CreatedAt) {
		t.Fatalf("bob's change: %+v", got)
	}
	before, by := w.cycleStamps(t, later)
	for _, body := range []string{`{}`, fmt.Sprintf(`{"name":"Later","start_date":%q,"end_date":%q}`, day(10), day(20))} {
		w.changeCycle(t, "bob", later, body, http.StatusOK)
		if at, who := w.cycleStamps(t, later); !at.Equal(before) || who != by {
			t.Errorf("a change of nothing (%s) wrote: %s by %s", body, at, who)
		}
	}
	// R4: the dates given alone, held to the cycle's.
	for _, tt := range []struct {
		id      uuid.UUID
		body    string
		refusal string
	}{
		{later, fmt.Sprintf(`{"start_date":%q}`, day(21)), "validation_failed start_date not_allowed"},
		{later, `{"end_date":null}`, "validation_failed end_date not_allowed"},
		{draft, fmt.Sprintf(`{"start_date":%q}`, day(30)), "validation_failed start_date not_allowed"},
		{later, fmt.Sprintf(`{"start_date":%q,"end_date":null}`, day(30)), "validation_failed end_date required"},
	} {
		w.refusedChange(t, "bob", tt.id, tt.body, http.StatusUnprocessableEntity, tt.refusal)
	}
	// Onto its own days, then onto Sprint's last.
	w.changeCycle(t, "bob", later, fmt.Sprintf(`{"start_date":%q,"end_date":%q}`, day(9), day(21)), http.StatusOK)
	w.refusedChange(t, "bob", later, fmt.Sprintf(`{"start_date":%q}`, day(2)), http.StatusConflict, "planning.cycle_dates_overlap")
	// R5: Old is completed.
	w.refusedChange(t, "bob", old, `{"name":" "}`, http.StatusUnprocessableEntity, "validation_failed name too_short")
	for _, body := range []string{`{"name":"Older"}`, `{"name":"Old"}`, fmt.Sprintf(`{"end_date":%q}`, day(-20))} {
		w.refusedChange(t, "bob", old, body, http.StatusConflict, "planning.cycle_completed")
	}
	decodeAnswer(t, w.changeCycle(t, "bob", old, `{"sort_order":-1}`, http.StatusOK), &got)
	if got.SortOrder != -1 || got.Status != "completed" {
		t.Fatalf("Old's new sort order: %+v", got)
	}
	// An archived cycle.
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE cycles SET archived_at = now() WHERE id = $1", old); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{`{}`, `{"sort_order":-2}`} {
		w.refusedChange(t, "bob", old, body, http.StatusConflict, "planning.archived")
	}
}

// The refusals before the cycle's own (M6 P1 spec R2): a cycle not there, or whose project the caller does not see,
// planning.cycle_not_found; a non-member who sees the project, a guest and an admin of the workspace who sees only his
// own work items there, forbidden; an archived project, then cycles turned off, 409.
func TestUpdateCycleRefusals(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	sprint := w.newCycle(t, "alice", `{"name":"Sprint"}`)
	w.refusedChange(t, "olga", sprint, `{"name":"A"}`, http.StatusNotFound, "planning.cycle_not_found")
	w.refusedChange(t, "alice", uuid.NewV7(), `{"name":"A"}`, http.StatusNotFound, "planning.cycle_not_found")
	// walt: no member of Web, then its guest, then acme's admin as well (adminGuest's two steps).
	w.refusedChange(t, "walt", sprint, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.makeMember(t, "walt", w.web, shared.RoleGuest)
	w.refusedChange(t, "walt", sprint, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	membership := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+membership.String(), `{"role":20}`, http.StatusOK)
	w.refusedChange(t, "walt", sprint, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedChange(t, "bob", sprint, `{"name":"A"}`, http.StatusConflict, "project.archived")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/unarchive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.refusedChange(t, "bob", sprint, `{"name":"A"}`, http.StatusConflict, "planning.cycles_disabled")
	w.refusedChange(t, "walt", sprint, `{"name":"A"}`, http.StatusForbidden, "forbidden")
	if c := w.getCycle(t, "bob", sprint); c.Name != "Sprint" {
		t.Fatalf("a refused change wrote: %+v", c)
	}
}
