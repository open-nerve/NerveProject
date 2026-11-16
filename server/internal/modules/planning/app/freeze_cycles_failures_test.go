package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// withFailingProject adds to w the project Mid, in UTC, whose one cycle to freeze, MidCycle, from the 20th to the 31st
// of October, fails once it wrote, and answers Mid.
func withFailingProject(t *testing.T, w *freezeWorld) uuid.UUID {
	t.Helper()
	mid, cycle := uuid.NewV7(), uuid.NewV7()
	w.names[mid], w.names[cycle] = "Mid", "MidCycle"
	w.projects[mid] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "UTC", CycleView: true}
	w.toFreeze[mid] = []domain.Cycle{{ID: cycle, ProjectID: mid, Dates: domain.Dates{Start: date(t, "2026-10-20"), End: date(t, "2026-10-31")}}}
	w.failOn, w.failure = cycle, errors.New("MidCycle's freeze fails")
	return mid
}

// cancelling is w's transactions, ending the run's context once the left-th of them has ended.
type cancelling struct {
	*freezeWorld
	left   int
	cancel context.CancelFunc
}

func (c *cancelling) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	err := c.freezeWorld.WithinTx(ctx, fn)
	if c.left--; c.left == 0 {
		c.cancel()
	}
	return err
}

// failingLocks is w's projects whose lock, logged, fails with errs' error for the project.
type failingLocks struct {
	*freezeWorld
	errs map[uuid.UUID]error
}

func (f failingLocks) Lock(ctx context.Context, id uuid.UUID) (ProjectFacts, bool, error) {
	f.log(ctx, "Lock "+f.names[id])
	return ProjectFacts{}, false, f.errs[id]
}

// A project's failure does not stop the run (M6 P8 spec 2.6): Shanghai, Mid, whose freeze fails, and UTC, each in a
// transaction of its own, its lock before its clock; Mid's is rolled back, Shanghai's and UTC's are committed and
// counted, and the run answers Mid's error whole, with Mid's id.
func TestFreezeCyclesGoesOnPastAFailedProject(t *testing.T) {
	w := newFreezeWorld(t)
	mid := withFailingProject(t, w)
	w.candidates = []uuid.UUID{freezeShanghai, mid, freezeUTC}
	n, err := w.execute(context.Background())
	if n != 3 || !errors.Is(err, w.failure) || !strings.Contains(err.Error(), mid.String()) {
		t.Fatalf("Execute = %d, %v; want Shanghai's 2 and UTC's 1, and Mid's failure with its id %s", n, err, mid)
	}
	want := []string{"Now outside tx", "FreezeDueProjects 2026-11-07 outside tx",
		"Begin", "Lock Shanghai", "Now", "CyclesToFreeze Shanghai 2026-11-08", "StateGroups Shanghai",
		"CycleItems Ended", "FreezeCycle Ended", "CycleItems Shelved", "FreezeCycle Shelved", "Commit",
		"Begin", "Lock Mid", "Now", "CyclesToFreeze Mid 2026-11-07", "StateGroups Mid", "CycleItems MidCycle", "FreezeCycle MidCycle",
		"Rollback",
		"Begin", "Lock UTC", "Now", "CyclesToFreeze UTC 2026-11-07", "StateGroups UTC", "CycleItems Old", "FreezeCycle Old", "Commit"}
	if !slices.Equal(w.calls, want) {
		t.Errorf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if _, ok := w.frozen[w.failOn]; ok || len(w.frozen) != 3 {
		t.Errorf("frozen = %v; want Ended's, Shelved's and Old's, not MidCycle's", w.frozen)
	}
}

// The run stops as its context ends, before the next project, and answers the failures before it with the context's
// error: Mid fails, Shanghai commits, the context ends, and UTC is not locked.
func TestFreezeCyclesStopsAtItsContextsEnd(t *testing.T) {
	w := newFreezeWorld(t)
	mid := withFailingProject(t, w)
	w.candidates = []uuid.UUID{mid, freezeShanghai, freezeUTC}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx := &cancelling{freezeWorld: w, left: 2, cancel: cancel}
	n, err := NewFreezeCycles(FreezeCyclesDeps{Projects: w, Groups: w, Cycles: w, Stats: w, Tx: tx, Clock: w}).Execute(ctx)
	if n != 2 || !errors.Is(err, w.failure) || !strings.Contains(err.Error(), mid.String()) || !errors.Is(err, context.Canceled) {
		t.Fatalf("Execute = %d, %v; want Shanghai's 2, Mid's failure with its id %s and the context's end", n, err, mid)
	}
	if slices.Contains(w.calls, "Lock UTC") || len(w.frozen) != 2 {
		t.Errorf("frozen %v, calls %q; want Shanghai's two committed and UTC not locked", w.frozen, w.calls)
	}
}

// Every project failing, as when the pool is lost midway, the run goes through them all, and its error keeps the first
// 5 projects' errors whole, with their ids, and counts the rest: the job's row keeps a bounded error (M6 P8 spec 2.6).
func TestFreezeCyclesBoundsItsError(t *testing.T) {
	w := newFreezeWorld(t)
	locks := failingLocks{freezeWorld: w, errs: map[uuid.UUID]error{}}
	w.candidates = nil
	for i := range 7 {
		id := uuid.NewV7()
		w.names[id] = fmt.Sprintf("P%d", i)
		w.candidates = append(w.candidates, id)
		locks.errs[id] = fmt.Errorf("P%d's lock fails", i)
	}
	n, err := NewFreezeCycles(FreezeCyclesDeps{Projects: locks, Groups: w, Cycles: w, Stats: w, Tx: w, Clock: w}).Execute(context.Background())
	if n != 0 || err == nil {
		t.Fatalf("Execute = %d, %v; want none frozen and the failures", n, err)
	}
	for i, id := range w.candidates {
		if !slices.Contains(w.calls, "Lock "+w.names[id]) {
			t.Errorf("P%d is not locked; want every project tried", i)
		}
		kept, named := errors.Is(err, locks.errs[id]), strings.Contains(err.Error(), id.String())
		if kept != (i < 5) || named != (i < 5) {
			t.Errorf("P%d's error in the run's %t, its id %t; want %t", i, kept, named, i < 5)
		}
	}
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) || len(joined.Unwrap()) != 6 || joined.Unwrap()[5].Error() != "and 2 more of the projects failed" {
		t.Errorf("the run's error %q; want the first 5 projects' errors and that 2 more failed", err)
	}
}
