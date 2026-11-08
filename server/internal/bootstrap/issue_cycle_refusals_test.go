package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// cycleWritten is what a refused write must leave as it was: the number of
// work items, of their cycle and module rows and of their activity rows,
// and the last change of each work item.
func (w issueWorld) cycleWritten(t *testing.T) string {
	t.Helper()
	var written string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT concat_ws(' ', (SELECT count(*) FROM issues), (SELECT count(*) FROM cycle_issues),
		(SELECT count(*) FROM module_issues), (SELECT count(*) FROM issue_activities),
		(SELECT string_agg(updated_at::text, ',' ORDER BY id) FROM issues))`).Scan(&written); err != nil {
		t.Fatal(err)
	}
	return written
}

// refusedJoin has the account name send method path with body, checks it
// is refused with status and the refusal (refusalOf: the code, then each
// field's), and that nothing was written.
func (w issueWorld) refusedJoin(t *testing.T, name, method, path, body string, status int, want string) {
	t.Helper()
	before := w.cycleWritten(t)
	if got := refusalOf(t, []byte(w.send(t, name, method, path, body, status)), false); got != want {
		t.Errorf("%s %s %s as %s = %s, want %s", method, path, body, name, got, want)
	}
	if after := w.cycleWritten(t); after != before {
		t.Errorf("%s %s %s as %s wrote: %s, was %s", method, path, body, name, after, before)
	}
}

// refusedShape is refusedJoin of a body the contract does not take, sent
// without checking it against the contract: refused as bad_request, with
// the refusal want, writing nothing.
func (w issueWorld) refusedShape(t *testing.T, name, method, path, body, want string) {
	t.Helper()
	before := w.cycleWritten(t)
	req := newRequest(t, method, w.base+path, w.tokens[name], []byte(body))
	res, answer := send(t, req)
	w.contract.CheckResponse(t, req, res)
	if got := refusalOf(t, answer, false); res.StatusCode != http.StatusBadRequest || got != want {
		t.Errorf("%s %s %s as %s = %d %s, want 400 %s", method, path, body, name, res.StatusCode, got, want)
	}
	if after := w.cycleWritten(t); after != before {
		t.Errorf("%s %s %s as %s wrote: %s, was %s", method, path, body, name, after, before)
	}
}

// The app refuses a work item's cycle as spec R2 orders it, on a creation
// and a change alike, writing nothing: a cycle ended on Shanghai's day,
// one archived, one deleted, another project's and none at all, each by
// its refusal, an unknown one with every other problem in one 422 (each
// cycle checked only when the work item joins it). Then each adjacent
// pair of the order: the body's shape before the checks without a read;
// those before the work item or the project looked up; the work item not
// seen before the role; the role before the archived project; the
// archived project before the archived work item; the archived work item
// before a change of nothing; a change of nothing before the cycles turned
// off, which refuse a change, leaving too, and a creation in a cycle; the
// cycles turned off before an archived cycle; an archived cycle before the
// 422; the 422 before the description's images; the images before an
// ended cycle.
func TestJoiningACycleIsRefusedInOrder(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	now := time.Now()
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-1), day(1)))
	ended := w.newCycle(t, "alice", datesBody("Ended", day(-30), day(-25)))
	shelved := w.newCycle(t, "alice", datesBody("Shelved", day(-24), day(-20)))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.turnCyclesOn(t, ops)
	var theirs cycleAnswer
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+ops.String()+"/cycles", `{"name":"Theirs"}`, http.StatusCreated), &theirs)
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","cycle_id":"`+sprint.String()+`"}`)
	create, change := "/api/v0/projects/"+w.web.String()+"/issues", "/api/v0/issues/"+item.String()
	in := func(cycle uuid.UUID, more string) string {
		return fmt.Sprintf(`{"name":"Fix","cycle_id":%q%s}`, cycle, more)
	}
	ghost, image := `,"label_ids":["`+uuid.NewV7().String()+`"]`, fmt.Sprintf(`,"description_html":%q`, issueImageHTML(uuid.NewV7()))
	unknown := "validation_failed cycle_id not_allowed"
	for _, path := range []string{create, change} {
		for _, tt := range []struct {
			cycle  uuid.UUID
			more   string
			status int
			want   string
		}{
			{ended, "", http.StatusConflict, "planning.cycle_completed"},
			{shelved, "", http.StatusConflict, "planning.archived"},
			{gone, "", http.StatusUnprocessableEntity, unknown},
			{theirs.ID, "", http.StatusUnprocessableEntity, unknown},
			{uuid.NewV7(), "", http.StatusUnprocessableEntity, unknown},
			{gone, ghost, http.StatusUnprocessableEntity, "validation_failed label_ids[0] not_allowed; cycle_id not_allowed"},
			{shelved, ghost, http.StatusConflict, "planning.archived"},
			{gone, image, http.StatusUnprocessableEntity, unknown},
			{ended, image, http.StatusNotFound, "asset.not_found"},
		} {
			method := map[string]string{create: http.MethodPost, change: http.MethodPatch}[path]
			w.refusedJoin(t, "bob", method, path, in(tt.cycle, tt.more), tt.status, tt.want)
		}
	}
	nowhere := "/api/v0/issues/" + uuid.NewV7().String()
	twice := `,"label_ids":["` + w.bug.String() + `","` + w.bug.String() + `"]`
	w.refusedShape(t, "bob", http.MethodPost, create, `{"name":" ","cycle_id":null}`, "bad_request cycle_id invalid_format")
	w.refusedShape(t, "bob", http.MethodPatch, change, `{"cycle_id":"Sprint"`+twice+`}`, "bad_request cycle_id invalid_format")
	w.refusedJoin(t, "bob", http.MethodPatch, nowhere, in(ended, twice), http.StatusUnprocessableEntity, "validation_failed label_ids[1] duplicate")
	w.refusedJoin(t, "olga", http.MethodPatch, change, in(ended, ""), http.StatusNotFound, "issue.not_found")
	w.refusedJoin(t, "walt", http.MethodPatch, change, in(ended, ""), http.StatusForbidden, "forbidden")

	// Web's cycles turned off: a change of the cycle, leaving it too, and a
	// creation in one, refused before an archived cycle; a change of
	// nothing is not refused, nor a creation in none.
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	for _, tt := range []struct{ method, path, body string }{{http.MethodPost, create, in(shelved, "")},
		{http.MethodPatch, change, in(shelved, "")}, {http.MethodPatch, change, `{"cycle_id":null}`}} {
		w.refusedJoin(t, "bob", tt.method, tt.path, tt.body, http.StatusConflict, "planning.cycles_disabled")
	}
	before := w.cycleWritten(t)
	w.send(t, "bob", http.MethodPatch, change, `{"cycle_id":"`+sprint.String()+`"}`, http.StatusOK)
	if after := w.cycleWritten(t); after != before {
		t.Errorf("a change of nothing wrote with cycles off: %s, was %s", after, before)
	}
	w.newWorkItemIn(t, w.web, `{"name":"Out of cycles"}`)
	// An archived work item before a change of nothing and the cycles
	// turned off; an archived project before it, and the role before that.
	done := w.newWorkItemIn(t, w.web, `{"name":"Done","state_id":"`+w.done.String()+`"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+done.String()+"/archive", "", http.StatusOK)
	for _, body := range []string{`{"cycle_id":null}`, in(sprint, "")} {
		w.refusedJoin(t, "bob", http.MethodPatch, "/api/v0/issues/"+done.String(), body, http.StatusConflict, "issue.archived")
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedJoin(t, "bob", http.MethodPatch, "/api/v0/issues/"+done.String(), in(sprint, ""), http.StatusConflict, "project.archived")
	w.refusedJoin(t, "bob", http.MethodPost, create, in(sprint, ""), http.StatusConflict, "project.archived")
	w.refusedJoin(t, "walt", http.MethodPost, create, in(sprint, ""), http.StatusForbidden, "forbidden")
}
