package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// A transfer that waits for Web's row past its lock timeout, as every write
// of the work-item side, is answered 503 server_busy, Retry-After: 1
// (busyTx; M6 P4 spec 2.3, A6), and writes nothing: Sprint is not frozen,
// a, b and d are as they were. Once the row is free, the same transfer
// moves a and b. The freeze job's lock timeout keeps PostgreSQL's error,
// and a failure keeps the projects committed before it
// (TestFreezeJobLockTimeoutPreservesPostgresError,
// TestFreezeJobPartialCommitAndRetry).
func TestATransferPastItsLockTimeoutIsBusy(t *testing.T) {
	r := newTransferRace(t)
	w := r.w
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	cfg := config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: config.IssueConfig{WriteLockTimeout: 300 * time.Millisecond}}
	probe := &requestProbe{TxManager: busyTx{tx: newIssueJobTx(w.pool, cfg)}}
	route := newPlanningRouteWith(t, w.pool, probe, r.clock, nil)
	var e exchange
	collect := requestBackground(t, w.sending(&e, route, "bob", http.MethodPost, transferPath(r.sprint), transferBody(r.next)),
		func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "projects")
	collect()
	w.refused(t, e, http.StatusServiceUnavailable, "server_busy")
	if got := e.rec.Header().Get("Retry-After"); got != "1" {
		t.Errorf("the busy transfer's Retry-After = %q; want 1", got)
	}
	if got := frozenOf(t, w.pool, r.sprint); got != nil {
		t.Errorf("Sprint frozen as %+v by a transfer that timed out", got)
	}
	r.wantTrails(t, nil)

	requestRelease(t, holder)
	var answer struct {
		Transferred int64 `json:"transferred_issues"`
	}
	decodeAnswer(t, route.answerWithin(t, w.contract, http.MethodPost, transferPath(r.sprint), w.ids["bob"], transferBody(r.next), http.StatusOK), &answer)
	if answer.Transferred != 2 {
		t.Errorf("the transfer once Web's row is free moved %d; want a and b", answer.Transferred)
	}
}

// heldTransfer is bob's transfer of the cycle from's work items to the
// cycle to through the planning module's route on the clock c, sent while
// a fixture holds Web's row: once it waits for that row, c moves on by
// two seconds, and the row is let go. It answers the transfer's exchange.
func heldTransfer(t *testing.T, w issueWorld, c *collaborationClock, from, to uuid.UUID) exchange {
	t.Helper()
	holder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", w.web)
	probe := requestTx(w.pool)
	var e exchange
	collect := requestBackground(t, w.sending(&e, newPlanningRouteWith(t, w.pool, probe, c, nil), "bob", http.MethodPost, transferPath(from),
		transferBody(to)), func() { requestRelease(t, holder) })
	requestBlocked(t, w.pool, probe, holder.Conn().PgConn().PID(), "projects")
	c.advance(2 * time.Second)
	requestRelease(t, holder)
	collect()
	return e
}

// A transfer reads its clock once it holds Web's row, its last lock, and
// judges and writes on that reading (M6 P4 spec 2.6, R7, A6; P1's
// update_cycle, P3a spec R2), on business clocks at the last second of
// Shanghai's day, Web's time zone, which the noon world started on, each
// moving on two seconds, to the next day there and still the same one in
// UTC, while its transfer waits for Web's row. Old, which a joined while it
// ran, ended yesterday; Sprint, which b joined, runs today alone; Next is
// ahead. A transfer of Old's to Sprint is refused 409
// planning.cycle_completed: Sprint has ended at its reading after the
// lock, not before it; nothing is frozen or moved. A transfer of Sprint's
// to Next goes on, Sprint ended then: Sprint frozen at that reading as
// alice read it before, b moved by bob at it.
func TestATransferReadsItsClockAfterItsLock(t *testing.T) {
	w, now := newNoonWorld(t)
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	day := func(days int) string { return shanghaiDay(t, now, days) }
	old := w.newCycle(t, "alice", datesBody("Old", day(-10), day(1)))
	a := w.newWorkItemBy(t, "alice", w.web, workItemBody("a", w.todo, nil, nil, `,"cycle_id":"`+old.String()+`"`))
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+old.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-1)+`"}`, http.StatusOK)
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(0), day(0)))
	b := w.newWorkItemBy(t, "alice", w.web, workItemBody("b", w.todo, nil, nil, `,"cycle_id":"`+sprint.String()+`"`))
	next := w.newCycle(t, "alice", datesBody("Next", day(5), day(10)))
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal(err)
	}
	today, err := time.ParseInLocation(time.DateOnly, day(0), loc)
	if err != nil {
		t.Fatal(err)
	}
	last := today.Add(24*time.Hour - time.Second).UTC()
	if after := last.Add(2 * time.Second); shanghaiDay(t, after, 0) != day(1) || after.UTC().Day() != last.UTC().Day() {
		t.Fatalf("the clocks' moments %s, %s must be on two days in Shanghai and one in UTC", last.UTC(), after.UTC())
	}
	n := cycleNames{w: w, cycles: map[uuid.UUID]string{old: "Old", sprint: "Sprint", next: "Next"}, moments: map[string]string{}}
	n.moment("after the lock", last.Add(2*time.Second))
	trails := map[uuid.UUID]itemTrail{a: n.trail(t, a), b: n.trail(t, b)}

	w.refused(t, heldTransfer(t, w, clockAt(last), old, sprint), http.StatusConflict, "planning.cycle_completed")
	if got := frozenOf(t, w.pool, old); got != nil {
		t.Errorf("Old frozen as %+v by a refused transfer", got)
	}
	for item, want := range trails {
		if got := n.trail(t, item); !reflect.DeepEqual(got, want) {
			t.Errorf("%s after the refused transfer:\n%+v\nwant\n%+v", item, got, want)
		}
	}

	live := w.cycleProgress(t, "alice", sprint).Live
	var answer struct {
		Transferred int64 `json:"transferred_issues"`
	}
	decodeAnswer(t, w.answered(t, heldTransfer(t, w, clockAt(last), sprint, next), http.StatusOK), &answer)
	if answer.Transferred != 1 {
		t.Errorf("the transfer of Sprint's moved %d; want b", answer.Transferred)
	}
	want := planningdomain.Frozen{Progress: live, FrozenAt: last.Add(2 * time.Second)}
	if got := frozenOf(t, w.pool, sprint); got == nil || !reflect.DeepEqual(*got, want) {
		t.Errorf("Sprint frozen as %+v; want %+v", got, want)
	}
	if got, want := n.trail(t, b), trails[b].moved("bob", "after the lock"); !reflect.DeepEqual(got, want) {
		t.Errorf("b after the transfer:\n%+v\nwant\n%+v", got, want)
	}
}
