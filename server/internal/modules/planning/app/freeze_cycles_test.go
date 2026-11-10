package app

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// freezeWorld is the freeze job's ports as fakes that log their calls in order, each inside a project's transaction
// or outside any, and keep a transaction's freezes only when it commits; a transaction asked for inside one joins it,
// as postgres.TxManager's does.
type freezeWorld struct {
	calls      []string
	names      map[uuid.UUID]string // the projects' and the cycles' names, for the log
	readings   []time.Time          // the clock's readings, in turn; the last one stays
	read       int
	candidates []uuid.UUID
	candidate  error                           // FreezeDueProjects' failure
	projects   map[uuid.UUID]ProjectFacts      // the projects Lock finds
	locking    error                           // Lock's failure
	toFreeze   map[uuid.UUID][]domain.Cycle    // what CyclesToFreeze answers, by project
	groups     map[uuid.UUID]domain.StateGroup // what StateGroups answers
	items      map[uuid.UUID][]domain.WorkItem // the work items of each cycle
	viewers    []*uuid.UUID                    // the viewer each CycleItems was asked for
	frozen     map[uuid.UUID]domain.Frozen     // the committed freezes, by cycle
	pending    map[uuid.UUID]domain.Frozen     // the open transaction's
	open       bool                            // whether a transaction is open
	failOn     uuid.UUID                       // the cycle whose freeze fails once it wrote
	failure    error
}

func (w *freezeWorld) log(ctx context.Context, s string) {
	if ctx.Value(inTx{}) == nil {
		s += " outside tx"
	}
	w.calls = append(w.calls, s)
}

func (w *freezeWorld) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if ctx.Value(inTx{}) != nil {
		return fn(ctx)
	}
	w.calls = append(w.calls, "Begin")
	w.pending, w.open = map[uuid.UUID]domain.Frozen{}, true
	err := fn(context.WithValue(ctx, inTx{}, true))
	w.open = false
	if err != nil {
		w.calls = append(w.calls, "Rollback")
		return err
	}
	for id, f := range w.pending {
		w.frozen[id] = f
	}
	w.calls = append(w.calls, "Commit")
	return nil
}

// Now is the clock's next reading, logged without a context: inside a transaction when one is open.
func (w *freezeWorld) Now() time.Time {
	s := "Now"
	if !w.open {
		s += " outside tx"
	}
	w.calls = append(w.calls, s)
	now := w.readings[min(w.read, len(w.readings)-1)]
	w.read++
	return now
}

func (w *freezeWorld) FreezeDueProjects(ctx context.Context, utcToday shared.Date) ([]uuid.UUID, error) {
	w.log(ctx, "FreezeDueProjects "+utcToday.String())
	return w.candidates, w.candidate
}

func (w *freezeWorld) Lock(ctx context.Context, id uuid.UUID) (ProjectFacts, bool, error) {
	w.log(ctx, "Lock "+w.names[id])
	if w.locking != nil {
		return ProjectFacts{}, false, w.locking
	}
	f, ok := w.projects[id]
	return f, ok, nil
}

func (w *freezeWorld) CyclesToFreeze(ctx context.Context, project uuid.UUID, today shared.Date) ([]domain.Cycle, error) {
	w.log(ctx, "CyclesToFreeze "+w.names[project]+" "+today.String())
	return w.toFreeze[project], nil
}

func (w *freezeWorld) StateGroups(ctx context.Context, project uuid.UUID) (map[uuid.UUID]domain.StateGroup, error) {
	w.log(ctx, "StateGroups "+w.names[project])
	return w.groups, nil
}

func (w *freezeWorld) CycleItems(ctx context.Context, _, cycle uuid.UUID, viewer *uuid.UUID) ([]domain.WorkItem, error) {
	w.log(ctx, "CycleItems "+w.names[cycle])
	w.viewers = append(w.viewers, viewer)
	return w.items[cycle], nil
}

func (w *freezeWorld) ModuleItems(ctx context.Context, _, _ uuid.UUID, _ *uuid.UUID) ([]domain.WorkItem, error) {
	w.log(ctx, "ModuleItems")
	return nil, nil
}

func (w *freezeWorld) CycleCounts(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.StateCount, error) {
	w.log(ctx, "CycleCounts")
	return nil, nil
}

func (w *freezeWorld) ModuleCounts(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.StateCount, error) {
	w.log(ctx, "ModuleCounts")
	return nil, nil
}

func (w *freezeWorld) CycleAssignees(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.CycleAssignee, error) {
	w.log(ctx, "CycleAssignees")
	return nil, nil
}

// FreezeCycle writes f when the cycle has no freeze, committed or in the open transaction, and then fails if it is
// failOn's.
func (w *freezeWorld) FreezeCycle(ctx context.Context, id uuid.UUID, f domain.Frozen) (bool, error) {
	w.log(ctx, "FreezeCycle "+w.names[id])
	_, committed := w.frozen[id]
	_, written := w.pending[id]
	if committed || written {
		return false, nil
	}
	w.pending[id] = f
	if id == w.failOn {
		return false, w.failure
	}
	return true, nil
}

func (w *freezeWorld) execute(ctx context.Context) (int, error) {
	return NewFreezeCycles(FreezeCyclesDeps{Projects: w, Groups: w, Cycles: w, Stats: w, Tx: w, Clock: w}).Execute(ctx)
}

var (
	freezeShanghai, freezeGone, freezeUTC = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	freezeEnded, freezeShelved, freezeOld = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	freezeBacklog, freezeDone             = uuid.NewV7(), uuid.NewV7()
	freezeAssignee, freezeLabel           = uuid.NewV7(), uuid.NewV7()
	// The clock's readings: the candidates', at 16:30 on the 7th of November 2026 in UTC, already the 8th in
	// Shanghai; then one a minute later for each project, after its lock.
	freezeReadings = []time.Time{time.Date(2026, 11, 7, 16, 30, 0, 0, time.UTC), time.Date(2026, 11, 7, 16, 31, 0, 0, time.UTC),
		time.Date(2026, 11, 7, 16, 32, 0, 0, time.UTC)}
)

// newFreezeWorld is the world of a run whose candidates are Shanghai, Gone, deleted, and UTC, archived with its
// cycles turned off. Shanghai's cycles to freeze on its day, the 8th, are Ended, from the 1st to the 7th, and
// Shelved, archived, from the 1st to the 7th of October; UTC's on its day, the 7th, is Old, from the 20th to the 31st
// of October. Ended has x, done at 16:10 on the 7th in UTC, the 8th in Shanghai, and y, in the backlog, assigned and
// labelled; Old has z, done on the 25th of October; Shelved has none.
func newFreezeWorld(t *testing.T) *freezeWorld {
	completed, done := freezeReadings[0].Add(-20*time.Minute), time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	creator := uuid.NewV7()
	cycle := func(id uuid.UUID, project uuid.UUID, start, end string, archived bool) domain.Cycle {
		c := domain.Cycle{ID: id, ProjectID: project, Dates: domain.Dates{Start: date(t, start), End: date(t, end)}, CreatedByID: &creator}
		if archived {
			c.ArchivedAt = &done
		}
		return c
	}
	return &freezeWorld{
		names: map[uuid.UUID]string{freezeShanghai: "Shanghai", freezeGone: "Gone", freezeUTC: "UTC", freezeEnded: "Ended",
			freezeShelved: "Shelved", freezeOld: "Old"},
		readings:   freezeReadings,
		candidates: []uuid.UUID{freezeShanghai, freezeGone, freezeUTC},
		projects: map[uuid.UUID]ProjectFacts{
			freezeShanghai: {WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true},
			freezeUTC:      {WorkspaceID: lockWorkspace, Timezone: "UTC", Archived: true},
		},
		toFreeze: map[uuid.UUID][]domain.Cycle{
			freezeShanghai: {cycle(freezeEnded, freezeShanghai, "2026-11-01", "2026-11-07", false),
				cycle(freezeShelved, freezeShanghai, "2026-10-01", "2026-10-07", true)},
			freezeUTC: {cycle(freezeOld, freezeUTC, "2026-10-20", "2026-10-31", false)},
		},
		groups: map[uuid.UUID]domain.StateGroup{freezeBacklog: domain.GroupBacklog, freezeDone: domain.GroupCompleted},
		items: map[uuid.UUID][]domain.WorkItem{
			freezeEnded: {{ID: uuid.NewV7(), StateID: freezeDone, CompletedAt: &completed},
				{ID: uuid.NewV7(), StateID: freezeBacklog, AssigneeIDs: []uuid.UUID{freezeAssignee}, LabelIDs: []uuid.UUID{freezeLabel}}},
			freezeOld: {{ID: uuid.NewV7(), StateID: freezeDone, CompletedAt: &done}},
		},
		frozen: map[uuid.UUID]domain.Frozen{},
	}
}

// frozenAt is the frozen progress of cycle c of w at the clock's reading at, in timezone: every work item's, on the
// day at falls on there.
func (w *freezeWorld) frozenAt(t *testing.T, c domain.Cycle, timezone string, at time.Time) domain.Frozen {
	t.Helper()
	today, err := domain.Today(at, timezone)
	if err != nil {
		t.Fatal(err)
	}
	p, err := domain.ProgressOf(domain.ProgressFacts{Items: w.items[c.ID], Groups: w.groups, Start: c.Dates.Start, End: c.Dates.End,
		Timezone: timezone, Today: today})
	if err != nil {
		t.Fatal(err)
	}
	return domain.Frozen{Progress: p, FrozenAt: at}
}

// The job reads its candidates without a lock, on the UTC day of the clock's first reading; then, for each, in a
// transaction of its own: the project's lock first, the clock read after it, the project's day in its time zone, the
// cycles to freeze on it, the states' groups once, and each cycle's work items, all of them, and its freeze, taken at
// that reading (M6 P4 spec 2.5, R5, R6). A deleted project is passed over before the clock; an archived project with
// its cycles off, and an archived cycle, are frozen as any other.
func TestFreezeCyclesLocksEachProjectBeforeItsClock(t *testing.T) {
	w := newFreezeWorld(t)
	n, err := w.execute(context.Background())
	if err != nil || n != 3 {
		t.Fatalf("Execute = %d, %v; want 3 frozen", n, err)
	}
	want := []string{"Now outside tx", "FreezeDueProjects 2026-11-07 outside tx",
		"Begin", "Lock Shanghai", "Now", "CyclesToFreeze Shanghai 2026-11-08", "StateGroups Shanghai",
		"CycleItems Ended", "FreezeCycle Ended", "CycleItems Shelved", "FreezeCycle Shelved", "Commit",
		"Begin", "Lock Gone", "Commit",
		"Begin", "Lock UTC", "Now", "CyclesToFreeze UTC 2026-11-07", "StateGroups UTC", "CycleItems Old", "FreezeCycle Old", "Commit"}
	if !slices.Equal(w.calls, want) {
		t.Errorf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if !slices.Equal(w.viewers, []*uuid.UUID{nil, nil, nil}) {
		t.Errorf("CycleItems' viewers = %v; want every work item, three times", w.viewers)
	}
	cycles := w.toFreeze[freezeShanghai]
	frozen := map[uuid.UUID]domain.Frozen{
		freezeEnded:   w.frozenAt(t, cycles[0], "Asia/Shanghai", freezeReadings[1]),
		freezeShelved: w.frozenAt(t, cycles[1], "Asia/Shanghai", freezeReadings[1]),
		freezeOld:     w.frozenAt(t, w.toFreeze[freezeUTC][0], "UTC", freezeReadings[2]),
	}
	if !reflect.DeepEqual(w.frozen, frozen) {
		t.Errorf("frozen\n%+v\nwant\n%+v", w.frozen, frozen)
	}
	// x, done on the 8th in Shanghai, after Ended's last day, is left on each of its days.
	if b := w.frozen[freezeEnded].Burndown; b == nil || *b.Points[len(b.Points)-1].Remaining != 2 {
		t.Errorf("Ended's burndown = %+v; want x not done by its last day in Shanghai", b)
	}
}

// A cycle whose freeze does not write, frozen already, is not counted and keeps its freeze; a project with no cycle
// to freeze on its day reads no states and no work items.
func TestFreezeCyclesCountsWhatItWrote(t *testing.T) {
	w := newFreezeWorld(t)
	first := domain.Frozen{Progress: domain.Progress{Counts: domain.Counts{Total: 9, Started: 9}}, FrozenAt: freezeReadings[0].AddDate(0, 0, -1)}
	w.frozen[freezeEnded] = first
	w.toFreeze[freezeUTC] = nil
	n, err := w.execute(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("Execute = %d, %v; want Shelved alone frozen", n, err)
	}
	if !reflect.DeepEqual(w.frozen[freezeEnded], first) {
		t.Errorf("Ended's freeze = %+v; want the first, %+v", w.frozen[freezeEnded], first)
	}
	if _, ok := w.frozen[freezeShelved]; !ok || len(w.frozen) != 2 {
		t.Errorf("frozen = %v; want Ended's first and Shelved's", w.frozen)
	}
	if want := []string{"Begin", "Lock UTC", "Now", "CyclesToFreeze UTC 2026-11-07", "Commit"}; !slices.Equal(w.calls[len(w.calls)-5:], want) {
		t.Errorf("UTC's calls end %q; want %q", w.calls, want)
	}
}

// A project's failure ends the run with it and the number frozen before it: the projects before it stay committed,
// its own writes are rolled back, and the projects after it wait for the next run (M6 P4 spec 2.5).
func TestFreezeCyclesKeepsTheProjectsBeforeAFailure(t *testing.T) {
	t.Run("a freeze", func(t *testing.T) {
		w := newFreezeWorld(t)
		second, after := uuid.NewV7(), uuid.NewV7()
		w.names[second], w.names[after] = "Second", "After"
		w.toFreeze[freezeUTC] = append(w.toFreeze[freezeUTC], domain.Cycle{ID: second, ProjectID: freezeUTC})
		w.candidates = append(w.candidates, after)
		w.projects[after] = ProjectFacts{Timezone: "UTC", CycleView: true}
		w.failOn, w.failure = second, errors.New("the second cycle's freeze fails")
		n, err := w.execute(context.Background())
		if n != 2 || !errors.Is(err, w.failure) {
			t.Fatalf("Execute = %d, %v; want Shanghai's 2 and the failure", n, err)
		}
		if len(w.frozen) != 2 || !slices.Contains(w.calls, "Rollback") || slices.Contains(w.calls, "Lock After") {
			t.Errorf("frozen %v, calls %q; want Shanghai's two committed, UTC's rolled back, After not locked", w.frozen, w.calls)
		}
		if _, ok := w.frozen[freezeOld]; ok {
			t.Error("Old, frozen in UTC's failed transaction, is committed")
		}
	})
	t.Run("a lock", func(t *testing.T) {
		w := newFreezeWorld(t)
		w.candidates = []uuid.UUID{freezeShanghai}
		w.locking = errors.New("lock timeout")
		if n, err := w.execute(context.Background()); n != 0 || !errors.Is(err, w.locking) || len(w.frozen) != 0 {
			t.Fatalf("Execute = %d, %v, frozen %v; want the lock's failure alone", n, err, w.frozen)
		}
		if want := []string{"Now outside tx", "FreezeDueProjects 2026-11-07 outside tx", "Begin", "Lock Shanghai", "Rollback"}; !slices.Equal(w.calls, want) {
			t.Errorf("calls %q; want %q: no clock before the lock", w.calls, want)
		}
	})
}

// Without candidates the run locks nothing; their read's failure, or a context done before it, is the run's.
func TestFreezeCyclesWithoutCandidates(t *testing.T) {
	w := newFreezeWorld(t)
	w.candidates = nil
	if n, err := w.execute(context.Background()); n != 0 || err != nil || len(w.calls) != 2 {
		t.Errorf("Execute = %d, %v, calls %q; want nothing locked", n, err, w.calls)
	}
	w = newFreezeWorld(t)
	w.candidate = errors.New("the candidates' read fails")
	if n, err := w.execute(context.Background()); n != 0 || !errors.Is(err, w.candidate) || slices.Contains(w.calls, "Begin") {
		t.Errorf("Execute = %d, %v, calls %q; want the read's failure", n, err, w.calls)
	}
	w = newFreezeWorld(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if n, err := w.execute(ctx); n != 0 || !errors.Is(err, context.Canceled) || len(w.calls) != 0 {
		t.Errorf("Execute = %d, %v, calls %q; want the context's end before anything", n, err, w.calls)
	}
}
