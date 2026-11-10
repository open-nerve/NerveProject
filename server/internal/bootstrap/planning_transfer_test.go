package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// transferPath is the path of a transfer of the cycle from's work items.
func transferPath(from uuid.UUID) string {
	return "/api/v0/cycles/" + from.String() + "/transfer-issues"
}

// transferBody is the body of a transfer to the cycle to.
func transferBody(to uuid.UUID) string { return `{"target_cycle_id":"` + to.String() + `"}` }

// transfer is how many work items the account name's transfer of the cycle
// from's to the cycle to moves, through the API.
func (w issueWorld) transfer(t *testing.T, name string, from, to uuid.UUID) int64 {
	t.Helper()
	var answer struct {
		Transferred int64 `json:"transferred_issues"`
	}
	decodeAnswer(t, w.send(t, name, http.MethodPost, transferPath(from), transferBody(to), http.StatusOK), &answer)
	return answer.Transferred
}

// refusedTransfer has the account name transfer the cycle from's work items
// with body and checks it is refused with status and the refusal
// (refusalOf: the code, then each field's).
func (w issueWorld) refusedTransfer(t *testing.T, name string, from uuid.UUID, body string, status int, refusal string) {
	t.Helper()
	if got := refusalOf(t, []byte(w.send(t, name, http.MethodPost, transferPath(from), body, status)), false); got != refusal {
		t.Errorf("transfer of %s as %s with %s = %s, want %s", from, name, body, got, refusal)
	}
}

// itemTrail is what the transfer test reads of a work item: its cycle
// rows, its activity and its last change (cycleNames).
type itemTrail struct {
	rows, activity []string
	audit          string
}

func (n cycleNames) trail(t *testing.T, item uuid.UUID) itemTrail {
	t.Helper()
	return itemTrail{rows: n.cycleRows(t, item), activity: n.cycleActivity(t, item), audit: n.audit(t, item)}
}

// moved is trail after a transfer from Sprint to Next by by at the moment
// named at (movedBetween).
func (trail itemTrail) moved(by, at string) itemTrail {
	return trail.movedBetween("Sprint", "Next", by, at)
}

// movedBetween is trail after a move from the cycle named from, its last
// row's, to the one named to, by by at the moment named at, a transfer's or
// a change's alike: from's row left, to's made, an updated row of activity
// naming both, the work item last changed then.
func (trail itemTrail) movedBetween(from, to, by, at string) itemTrail {
	rows := append([]string{}, trail.rows...)
	rows[len(rows)-1] += ", left by " + by + " at " + at + ", written at " + at
	return itemTrail{rows: append(rows, to+" made by "+by+" at "+at),
		activity: append(append([]string{}, trail.activity...), `cycles updated "`+from+`"→"`+to+`" `+from+`→`+to+` "updated cycle from " by `+by+` at `+at),
		audit:    "by " + by + " at " + at}
}

// The app transfers a cycle's unfinished work items as the contract says
// (M6 P4 spec 2.3, 2.6, R7, R8), on the business clock at noon in Shanghai,
// Web's time zone. Sprint, which had its work items while it ran: a in the
// backlog, b in Todo, c in progress, d done, e cancelled, f archived in
// Todo, g deleted from Todo. It ends yesterday; bob transfers to Next, which
// is ahead. a, b and c move, each its row of Sprint left and one of Next
// made, one updated row of activity naming both, last changed by bob, at
// the transfer's moment; d, e, f and g stay as they were. Sprint is frozen
// first, at that moment, as alice read its live progress before, its last
// change its own; it now counts d and e, and Next a, b and c. Again, none is
// left: 0, nothing written. d put back in Todo, alice's transfer moves it
// alone; Sprint's freeze stays the first.
func TestTransferCycleIssuesThroughTheApp(t *testing.T) {
	w, now := newNoonWorld(t)
	// The app's run at start finds nothing to freeze, and none runs again.
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-10), day(1)))
	next := w.newCycle(t, "alice", datesBody("Next", day(5), day(10)))
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{sprint: "Sprint", next: "Next"}, moments: map[string]string{}}
	backlog, started, cancelled := stateID(t, w.pool, w.web, "Backlog"), stateID(t, w.pool, w.web, "In Progress"), stateID(t, w.pool, w.web, "Cancelled")
	item := func(name string, state uuid.UUID) uuid.UUID {
		made := w.joinWrite(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues",
			workItemBody(name, state, nil, nil, `,"cycle_id":"`+sprint.String()+`"`), http.StatusCreated)
		n.moment("made "+name, made.CreatedAt)
		return made.ID
	}
	a, b, c, d, e, f, g := item("a", backlog), item("b", w.todo), item("c", started), item("d", w.done), item("e", cancelled), item("f", w.todo),
		item("g", w.todo)
	listSeedSQL(t, w, "UPDATE issues SET archived_at = now() WHERE id = $1", f)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+g.String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+sprint.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-1)+`"}`, http.StatusOK)

	items := map[string]uuid.UUID{"a": a, "b": b, "c": c, "d": d, "e": e, "f": f, "g": g}
	before := map[string]itemTrail{}
	for name, id := range items {
		before[name] = n.trail(t, id)
	}
	stamped, by := w.cycleStamps(t, sprint)
	live := w.cycleProgress(t, "alice", sprint).Live
	if moved := w.transfer(t, "bob", sprint, next); moved != 3 {
		t.Fatalf("bob's transfer moved %d; want a, b and c", moved)
	}
	frozen := frozenOf(t, w.pool, sprint)
	if frozen == nil || !reflect.DeepEqual(*frozen, planningdomain.Frozen{Progress: live, FrozenAt: frozen.FrozenAt}) || frozen.FrozenAt.Before(now) {
		t.Fatalf("Sprint frozen as %+v; want its live progress before the move, %+v, at the transfer's moment", frozen, live)
	}
	n.moment("transferred", frozen.FrozenAt)
	after := map[string]itemTrail{}
	for name, id := range items {
		after[name] = n.trail(t, id)
		want := before[name]
		if name == "a" || name == "b" || name == "c" {
			want = want.moved("bob", "transferred")
		}
		if !reflect.DeepEqual(after[name], want) {
			t.Errorf("%s after the transfer:\n%+v\nwant\n%+v", name, after[name], want)
		}
	}
	if at, who := w.cycleStamps(t, sprint); !at.Equal(stamped) || who != by {
		t.Errorf("Sprint last changed by %s at %s; want as it was, by %s at %s", who, at, by, stamped)
	}
	read := w.cycleProgress(t, "alice", sprint)
	if read.Live.Counts != (planningdomain.Counts{Total: 2, Completed: 1, Cancelled: 1}) || read.Snapshot == nil || !reflect.DeepEqual(*read.Snapshot, *frozen) {
		t.Errorf("Sprint reads %+v; want d and e live, and the freeze", read)
	}
	if got := w.cycleProgress(t, "alice", next).Live.Counts; got != (planningdomain.Counts{Total: 3, Backlog: 1, Unstarted: 1, Started: 1}) {
		t.Errorf("Next counts %+v; want a, b and c", got)
	}

	// None is left to move: 0, and nothing written.
	if moved := w.transfer(t, "bob", sprint, next); moved != 0 {
		t.Fatalf("a second transfer moved %d; want none", moved)
	}
	for name, id := range items {
		if got := n.trail(t, id); !reflect.DeepEqual(got, after[name]) {
			t.Errorf("%s after a second transfer:\n%+v\nwant\n%+v", name, got, after[name])
		}
	}

	// d back in Todo moves alone, by alice; the freeze stays the first.
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+d.String(), `{"state_id":"`+w.todo.String()+`"}`, http.StatusOK)
	reopened := n.trail(t, d)
	if moved := w.transfer(t, "alice", sprint, next); moved != 1 {
		t.Fatalf("the third transfer moved %d; want d", moved)
	}
	var again time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at FROM issues WHERE id = $1", d).Scan(&again); err != nil {
		t.Fatal(err)
	}
	n.moment("again", again)
	if got, want := n.trail(t, d), reopened.moved("alice", "again"); !reflect.DeepEqual(got, want) {
		t.Errorf("d after the third transfer:\n%+v\nwant\n%+v", got, want)
	}
	if got := frozenOf(t, w.pool, sprint); !reflect.DeepEqual(got, frozen) {
		t.Errorf("Sprint's freeze after the third transfer %+v; want the first, %+v", got, frozen)
	}
}

// A cycle the job froze keeps that freeze through a transfer of its work
// items (M6 P4 spec R5, R7): Ended, which a, in Todo, and b, done, were in
// while it ran, is frozen by the job at noon; bob's transfer to Next moves a
// and leaves the freeze as the job wrote it.
func TestTransferKeepsTheJobsFreeze(t *testing.T) {
	w, now := newNoonWorld(t)
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	day := func(days int) string { return shanghaiDay(t, now, days) }
	ended := w.newCycle(t, "alice", datesBody("Ended", day(-10), day(1)))
	next := w.newCycle(t, "alice", datesBody("Next", day(5), day(10)))
	inEnded := `,"cycle_id":"` + ended.String() + `"`
	a := w.newWorkItemBy(t, "alice", w.web, workItemBody("a", w.todo, nil, nil, inEnded))
	w.newWorkItemBy(t, "alice", w.web, workItemBody("b", w.done, nil, nil, inEnded))
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+ended.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-1)+`"}`, http.StatusOK)
	if frozen, err := freezeCyclesOn(w.pool, clocktest.At(now)).Execute(pgtest.Soon(t)); err != nil || frozen != 1 {
		t.Fatalf("the job froze %d, %v; want Ended", frozen, err)
	}
	job := frozenOf(t, w.pool, ended)
	if job == nil || !job.FrozenAt.Equal(now) || job.Total != 2 {
		t.Fatalf("Ended frozen as %+v; want a and b, at the job's moment", job)
	}
	if moved := w.transfer(t, "bob", ended, next); moved != 1 {
		t.Fatalf("bob's transfer moved %d; want a", moved)
	}
	if got := frozenOf(t, w.pool, ended); !reflect.DeepEqual(got, job) {
		t.Errorf("Ended's freeze after the transfer %+v; want the job's, %+v", got, job)
	}
	if ids := queryIDs(t, w.pool, "SELECT issue_id FROM cycle_issues WHERE cycle_id = $1 AND deleted_at IS NULL", next); len(ids) != 1 || ids[0] != a {
		t.Errorf("Next has %v; want a", ids)
	}
}
