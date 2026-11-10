package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	planningpg "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	planningdomain "github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of a transfer and of the freeze job (M6 P4 spec 2.5,
// 2.6, A2, A6) with each other and with the writes on what a transfer
// moves, on the real database: each write as bootstrap wires it, the first
// held open past its writes (heldTx), the second sent once it is held and
// let go once it waits for Web's row, which each takes FOR NO KEY UPDATE
// first (serializeAtAfter, the wait attributed by its PID).

// transferRace is one order of such an interleaving on the noon world
// (newNoonWorld), once the app's runs at start are done: Web's cycles on;
// Sprint, which a and b, in Todo, and d, done, joined while it ran, all
// alice's, ended yesterday on Shanghai's day; Next, ahead; Other, ahead
// after it. Both writes read clock: it stands at first, past the
// fixture's moments, which the first write reads, and moves on to second
// while the second write waits for Web's row, which it reads after it. live
// is Sprint's progress as alice reads it before the writes, before their
// trails (names: the accounts, the cycles, Todo and Done, and the two
// moments); frozen, how many cycles the job froze, when it ran.
type transferRace struct {
	w                   issueWorld
	clock               *collaborationClock
	first, second       time.Time
	sprint, next, other uuid.UUID
	items               map[string]uuid.UUID
	names               cycleNames
	before              map[string]itemTrail
	live                planningdomain.Progress
	frozen              int
}

// newTransferRace prepares the world of one order of an interleaving.
func newTransferRace(t *testing.T) *transferRace {
	t.Helper()
	w, now := newNoonWorld(t)
	started := time.Now()
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	day := func(days int) string { return shanghaiDay(t, now, days) }
	r := &transferRace{w: w, sprint: w.newCycle(t, "alice", datesBody("Sprint", day(-10), day(1))),
		next: w.newCycle(t, "alice", datesBody("Next", day(5), day(10))), other: w.newCycle(t, "alice", datesBody("Other", day(12), day(15)))}
	in := `,"cycle_id":"` + r.sprint.String() + `"`
	r.items = map[string]uuid.UUID{"a": w.newWorkItemBy(t, "alice", w.web, workItemBody("a", w.todo, nil, nil, in)),
		"b": w.newWorkItemBy(t, "alice", w.web, workItemBody("b", w.todo, nil, nil, in)),
		"d": w.newWorkItemBy(t, "alice", w.web, workItemBody("d", w.done, nil, nil, in))}
	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+r.sprint.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-1)+`"}`, http.StatusOK)
	r.names = cycleNames{w: w, cycles: map[uuid.UUID]string{r.sprint: "Sprint", r.next: "Next", r.other: "Other", w.todo: "Todo", w.done: "Done"},
		moments: map[string]string{}}
	r.before = map[string]itemTrail{}
	for name, id := range r.items {
		r.before[name] = r.names.trail(t, id)
	}
	r.live = w.cycleProgress(t, "alice", r.sprint).Live
	r.first = now.Add(time.Since(started) + time.Second).UTC().Truncate(time.Microsecond)
	r.second = r.first.Add(time.Second)
	r.clock = clockAt(r.first)
	r.names.moment("first", r.first)
	r.names.moment("second", r.second)
	return r
}

// transferRaces runs a, then b, and b, then a, each on a transferRace of
// its own, the second waiting for Web's row while the clock moves on to
// second, and checks each order with check. aName and bName name the
// writes in the subtests' names.
func transferRaces(t *testing.T, aName string, a func(*transferRace) racingWrite, bName string, b func(*transferRace) racingWrite,
	check func(t *testing.T, r *transferRace, aFirst bool, ea, eb exchange)) {
	for _, aFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: aName, false: bName}[aFirst]+" first", func(t *testing.T) {
			r := newTransferRace(t)
			var ea, eb exchange
			advance := func() { r.clock.advance(r.second.Sub(r.first)) }
			if aFirst {
				r.w.serializeAtAfter(t, "projects", a(r), b(r), &ea, &eb, advance)
			} else {
				r.w.serializeAtAfter(t, "projects", b(r), a(r), &eb, &ea, advance)
			}
			check(t, r, aFirst, ea, eb)
		})
	}
}

// transfers is bob's transfer of Sprint's work items to Next, through the
// planning module's route on r's clock.
func (r *transferRace) transfers() racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newPlanningRouteWith(t, w.pool, tx, r.clock, nil), "bob", http.MethodPost, transferPath(r.sprint), transferBody(r.next))
	}
}

// deletesNext is alice's deletion of Next through the planning module's
// route on r's clock, published to the issue module's subscribers in its
// transaction.
func (r *transferRace) deletesNext() racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newPlanningRouteWith(t, w.pool, tx, r.clock, nil), "alice", http.MethodDelete, "/api/v0/cycles/"+r.next.String(), "")
	}
}

// freezes is a run of the freeze job's use case, as planning.New wires it,
// on r's clock, in tx's transactions; r.frozen keeps how many it froze.
func (r *transferRace) freezes() racingWrite {
	return func(_ *testing.T, w issueWorld, tx shared.TxManager, _ *exchange) func() error {
		job := freezeCyclesWith(w.pool, planningpg.New(w.pool), tx, r.clock)
		return func() error {
			var err error
			r.frozen, err = job.Execute(context.Background())
			return err
		}
	}
}

// changes is alice's change of the work item named item with body, through
// the issue module's route on r's clock.
func (r *transferRace) changes(item, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRouteWithClock(t, w.pool, tx, r.clock), "alice", http.MethodPatch, "/api/v0/issues/"+r.items[item].String(), body)
	}
}

// completesA is alice's change of a into Done; movesA, of a into Other.
func (r *transferRace) completesA() racingWrite {
	return r.changes("a", `{"state_id":"`+r.w.done.String()+`"}`)
}

func (r *transferRace) movesA() racingWrite {
	return r.changes("a", `{"cycle_id":"`+r.other.String()+`"}`)
}

// transferred checks that the transfer e answered 200, n work items moved.
func (r *transferRace) transferred(t *testing.T, e exchange, n int64) {
	t.Helper()
	var answer struct {
		Transferred int64 `json:"transferred_issues"`
	}
	decodeAnswer(t, r.w.answered(t, e, http.StatusOK), &answer)
	if answer.Transferred != n {
		t.Errorf("the transfer moved %d; want %d", answer.Transferred, n)
	}
}

// wantFrozen checks that Sprint is frozen as p, at.
func (r *transferRace) wantFrozen(t *testing.T, p planningdomain.Progress, at time.Time) {
	t.Helper()
	want := planningdomain.Frozen{Progress: p, FrozenAt: at}
	if got := frozenOf(t, r.w.pool, r.sprint); got == nil || !reflect.DeepEqual(*got, want) {
		t.Errorf("Sprint frozen as %+v; want %+v", got, want)
	}
}

// wantFrozenCounts checks that Sprint is frozen with the counts c, at.
func (r *transferRace) wantFrozenCounts(t *testing.T, c planningdomain.Counts, at time.Time) {
	t.Helper()
	if got := frozenOf(t, r.w.pool, r.sprint); got == nil || got.Counts != c || !got.FrozenAt.Equal(at) {
		t.Errorf("Sprint frozen as %+v; want %+v at %s", got, c, at)
	}
}

// wantTrails checks each work item's trail against want's, by its name, and
// against its trail before the writes when want has none.
func (r *transferRace) wantTrails(t *testing.T, want map[string]itemTrail) {
	t.Helper()
	for name, id := range r.items {
		w, ok := want[name]
		if !ok {
			w = r.before[name]
		}
		if got := r.names.trail(t, id); !reflect.DeepEqual(got, w) {
			t.Errorf("%s:\n%+v\nwant\n%+v", name, got, w)
		}
	}
}

// completed is trail after a change of the work item from Todo to Done by
// by at the moment named at.
func (trail itemTrail) completed(by, at string) itemTrail {
	return itemTrail{rows: trail.rows, activity: append(append([]string{}, trail.activity...),
		`state updated "Todo"→"Done" Todo→Done "updated the state to" by `+by+` at `+at), audit: "by " + by + " at " + at}
}

// left is trail after the deletion of its cycle, named cycle, by by at the
// moment named at: its row left, a deleted row of activity naming it.
func (trail itemTrail) left(cycle, by, at string) itemTrail {
	rows := append([]string{}, trail.rows...)
	rows[len(rows)-1] += ", left by " + by + " at " + at + ", written at " + at
	return itemTrail{rows: rows, activity: append(append([]string{}, trail.activity...),
		`cycles deleted "`+cycle+`"→"" `+cycle+`→- "removed this work item from cycle " by `+by+` at `+at), audit: "by " + by + " at " + at}
}

// The freeze job and a transfer of Sprint's work items serialize on Web's
// row (M6 P4 spec 2.5, 2.6, R5, A2), in both orders. Sprint is frozen once,
// by the first, at its moment, with the numbers it had before the move, as
// alice read them: the job, first, froze it and the transfer, waiting,
// found it frozen and kept that freeze; or the transfer, first, froze it,
// and the job, waiting, found nothing to freeze. Either way the transfer
// moved a and b, by bob at its moment, and left d.
func TestTheFreezeJobAndATransferSerialize(t *testing.T) {
	transferRaces(t, "job", (*transferRace).freezes, "transfer", (*transferRace).transfers,
		func(t *testing.T, r *transferRace, jobFirst bool, _, transfer exchange) {
			r.transferred(t, transfer, 2)
			moved, froze := "first", 0
			if jobFirst {
				moved, froze = "second", 1
			}
			if r.frozen != froze {
				t.Errorf("the job froze %d; want %d", r.frozen, froze)
			}
			r.wantFrozen(t, r.live, r.first)
			r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].moved("bob", moved), "b": r.before["b"].moved("bob", moved)})
		})
}

// A transfer of Sprint's work items and alice's completion of a, one of
// them, serialize on Web's row (M6 P4 spec 2.6, R7, A6), in both orders.
// Completed first, a stays in Sprint, done, and the transfer, waiting,
// moves b alone, Sprint frozen at its moment with a done; transferred
// first, a and b move to Next, Sprint frozen as alice read it before, and
// the completion, waiting, completes a in Next.
func TestATransferAndACompletionOfItsWorkItemSerialize(t *testing.T) {
	transferRaces(t, "transfer", (*transferRace).transfers, "completion", (*transferRace).completesA,
		func(t *testing.T, r *transferRace, transferFirst bool, transfer, completion exchange) {
			r.w.answered(t, completion, http.StatusOK)
			if !transferFirst {
				r.transferred(t, transfer, 1)
				r.wantFrozenCounts(t, planningdomain.Counts{Total: 3, Unstarted: 1, Completed: 2}, r.second)
				r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].completed("alice", "first"), "b": r.before["b"].moved("bob", "second")})
				return
			}
			r.transferred(t, transfer, 2)
			r.wantFrozen(t, r.live, r.first)
			r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].moved("bob", "first").completed("alice", "second"),
				"b": r.before["b"].moved("bob", "first")})
			if got := r.w.cycleProgress(t, "alice", r.next).Live.Counts; got != (planningdomain.Counts{Total: 2, Unstarted: 1, Completed: 1}) {
				t.Errorf("Next counts %+v; want b in Todo and a done", got)
			}
		})
}

// A transfer of Sprint's work items and alice's move of a, one of them, to
// Other serialize on Web's row (M6 P4 spec 2.6, R8, A6; P3a spec R4), in
// both orders, each move one row of activity. Moved first, a is in Other
// and the transfer, waiting, moves b alone, Sprint frozen at its moment
// without a; transferred first, a and b move to Next, Sprint frozen as
// alice read it before, and the move, waiting, moves a on from Next.
func TestATransferAndAMoveOfItsWorkItemSerialize(t *testing.T) {
	transferRaces(t, "transfer", (*transferRace).transfers, "move", (*transferRace).movesA,
		func(t *testing.T, r *transferRace, transferFirst bool, transfer, move exchange) {
			r.w.answered(t, move, http.StatusOK)
			if !transferFirst {
				r.transferred(t, transfer, 1)
				r.wantFrozenCounts(t, planningdomain.Counts{Total: 2, Unstarted: 1, Completed: 1}, r.second)
				r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].movedBetween("Sprint", "Other", "alice", "first"),
					"b": r.before["b"].moved("bob", "second")})
				return
			}
			r.transferred(t, transfer, 2)
			r.wantFrozen(t, r.live, r.first)
			r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].moved("bob", "first").movedBetween("Next", "Other", "alice", "second"),
				"b": r.before["b"].moved("bob", "first")})
		})
}

// A transfer of Sprint's work items and alice's deletion of Next, its
// target, serialize on Web's row (M6 P4 spec 2.3, 2.6, A6; P3a spec R6), in
// both orders. Deleted first, the transfer, waiting, reads Next again under
// the lock and finds it gone: 422 at target_cycle_id, nothing frozen or
// moved. Transferred first, a and b move to Next, Sprint frozen as alice
// read it before, and the deletion, waiting, takes them out of Next, one
// row of activity each, by alice at its moment.
func TestATransferAndTheDeletionOfItsTargetSerialize(t *testing.T) {
	transferRaces(t, "transfer", (*transferRace).transfers, "deletion", (*transferRace).deletesNext,
		func(t *testing.T, r *transferRace, transferFirst bool, transfer, deletion exchange) {
			r.w.answered(t, deletion, http.StatusNoContent)
			if !transferFirst {
				r.w.refused(t, transfer, http.StatusUnprocessableEntity, "validation_failed target_cycle_id not_allowed")
				if got := frozenOf(t, r.w.pool, r.sprint); got != nil {
					t.Errorf("Sprint frozen as %+v by a refused transfer", got)
				}
				r.wantTrails(t, nil)
				return
			}
			r.transferred(t, transfer, 2)
			r.wantFrozen(t, r.live, r.first)
			r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].moved("bob", "first").left("Next", "alice", "second"),
				"b": r.before["b"].moved("bob", "first").left("Next", "alice", "second")})
		})
}

// The freeze job and alice's completion of a, in Sprint, which has ended
// and is not frozen, serialize on Web's row (M6 P4 spec 2.5, A2, A6), in
// both orders, and the freeze is of whichever came first: the job first,
// a in Todo, as alice read Sprint before; the completion first, a done, as
// alice reads Sprint after both.
func TestTheFreezeJobAndACompletionSerialize(t *testing.T) {
	transferRaces(t, "job", (*transferRace).freezes, "completion", (*transferRace).completesA,
		func(t *testing.T, r *transferRace, jobFirst bool, _, completion exchange) {
			r.w.answered(t, completion, http.StatusOK)
			if r.frozen != 1 {
				t.Errorf("the job froze %d; want Sprint", r.frozen)
			}
			live := r.w.cycleProgress(t, "alice", r.sprint).Live
			if live.Counts != (planningdomain.Counts{Total: 3, Unstarted: 1, Completed: 2}) {
				t.Errorf("Sprint counts %+v; want a done", live.Counts)
			}
			completed := "first"
			if jobFirst {
				completed = "second"
				r.wantFrozen(t, r.live, r.first)
			} else {
				r.wantFrozen(t, live, r.second)
			}
			r.wantTrails(t, map[string]itemTrail{"a": r.before["a"].completed("alice", completed)})
		})
}
