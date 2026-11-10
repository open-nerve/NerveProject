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

// CycleItems is the lock world's IssueStats: the work items of the cycle, when it is of project, whoever views them
// (the issue module filters them by viewer, whom the world records).
func (w *lockWorld) CycleItems(ctx context.Context, project, cycle uuid.UUID, viewer *uuid.UUID) ([]domain.WorkItem, error) {
	w.log(ctx, "CycleItems")
	w.viewers = append(w.viewers, viewer)
	if c, ok := w.cycles[cycle]; !ok || c.ProjectID != project {
		return nil, nil
	}
	return w.items[cycle], nil
}

// ModuleItems is CycleItems of a module.
func (w *lockWorld) ModuleItems(ctx context.Context, project, module uuid.UUID, viewer *uuid.UUID) ([]domain.WorkItem, error) {
	w.log(ctx, "ModuleItems")
	w.viewers = append(w.viewers, viewer)
	if m, ok := w.modules[module]; !ok || m.ProjectID != project {
		return nil, nil
	}
	return w.items[module], nil
}

func (w *lockWorld) CycleCounts(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.StateCount, error) {
	w.log(ctx, "CycleCounts")
	return nil, nil
}

func (w *lockWorld) ModuleCounts(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.StateCount, error) {
	w.log(ctx, "ModuleCounts")
	return nil, nil
}

func (w *lockWorld) CycleAssignees(ctx context.Context, _ uuid.UUID, _ []uuid.UUID, _ *uuid.UUID) ([]domain.CycleAssignee, error) {
	w.log(ctx, "CycleAssignees")
	return nil, nil
}

// CycleSnapshot is the lock world's frozen progress of the cycle id, nil for none.
func (w *lockWorld) CycleSnapshot(ctx context.Context, id uuid.UUID) (*domain.Frozen, error) {
	w.log(ctx, "CycleSnapshot")
	return w.snapshots[id], nil
}

func (w *lockWorld) cycleProgress(actor shared.Actor) (domain.CycleProgress, error) {
	uc := NewGetCycleProgress(GetCycleProgressDeps{Reads: w.reads(), Groups: w, Stats: w, Snapshots: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), actor), lockCycle)
}

func (w *lockWorld) moduleProgress(actor shared.Actor) (domain.Progress, error) {
	uc := NewGetModuleProgress(GetModuleProgressDeps{Reads: w.reads(), Groups: w, Stats: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), actor), lockModule)
}

var (
	progressBacklog, progressDone, progressAssignee = uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	// progressFrozen is the frozen progress of the progress world's cycle: other numbers than its live ones.
	progressFrozen = domain.Frozen{Progress: domain.Progress{Counts: domain.Counts{Total: 5, Started: 5},
		Assignees: []domain.AssigneeDistribution{}, Labels: []domain.LabelDistribution{}}, FrozenAt: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)}
)

// newProgressWorld is the module world in Shanghai, at 01:00 on the 8th of November 2026 there, the 7th in UTC; its
// project has a backlog and a completed state. Its cycle runs from the 8th to the 14th, frozen as progressFrozen, and
// its module from the 8th to the 14th; each has two work items: one in the backlog, assigned, and one completed at
// 00:30 on the 8th in Shanghai, the 7th in UTC.
func newProgressWorld(t *testing.T) *lockWorld {
	w := newModuleWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true}
	c := w.cycles[lockCycle]
	c.Dates = domain.Dates{Start: date(t, "2026-11-08"), End: date(t, "2026-11-14")}
	w.cycles[lockCycle] = c
	m := w.modules[lockModule]
	m.StartDate, m.TargetDate = date(t, "2026-11-08"), date(t, "2026-11-14")
	w.modules[lockModule] = m
	w.now = time.Date(2026, 11, 7, 17, 0, 0, 0, time.UTC)
	w.groups = map[uuid.UUID]domain.StateGroup{progressBacklog: domain.GroupBacklog, progressDone: domain.GroupCompleted}
	completed := time.Date(2026, 11, 7, 16, 30, 0, 0, time.UTC)
	items := []domain.WorkItem{{ID: uuid.NewV7(), StateID: progressBacklog, AssigneeIDs: []uuid.UUID{progressAssignee}},
		{ID: uuid.NewV7(), StateID: progressDone, CompletedAt: &completed}}
	w.items = map[uuid.UUID][]domain.WorkItem{lockCycle: items, lockModule: items}
	w.snapshots = map[uuid.UUID]*domain.Frozen{lockCycle: &progressFrozen}
	return w
}

// progressWorldLive is the live progress of the progress world's cycle and module on the 8th in Shanghai: one work
// item of two completed that day, the days after it without a number.
func progressWorldLive(t *testing.T) domain.Progress {
	one := int64(1)
	points := []domain.BurndownPoint{{Date: *date(t, "2026-11-08"), Remaining: &one}}
	for _, d := range []string{"2026-11-09", "2026-11-10", "2026-11-11", "2026-11-12", "2026-11-13", "2026-11-14"} {
		points = append(points, domain.BurndownPoint{Date: *date(t, d)})
	}
	return domain.Progress{Counts: domain.Counts{Total: 2, Backlog: 1, Completed: 1},
		Assignees: []domain.AssigneeDistribution{{AssigneeID: &progressAssignee, Tally: domain.Tally{Total: 1, Pending: 1}},
			{Tally: domain.Tally{Total: 1, Completed: 1}}},
		Labels:   []domain.LabelDistribution{{Tally: domain.Tally{Total: 2, Completed: 1, Pending: 1}}},
		Burndown: &domain.Burndown{StartDate: *date(t, "2026-11-08"), EndDate: *date(t, "2026-11-14"), Scope: 2, Points: points}}
}

// A cycle's progress is read in its own read-only snapshot (Reads, cycle.read): the cycle, its project's facts and
// the decision, then its project's states' groups, its work items of every account's, its frozen progress, and last
// the clock, all inside the transaction. Its status and its burndown are on the project's day, the 8th in Shanghai,
// not UTC's 7th. Not frozen, it has none.
func TestGetCycleProgress(t *testing.T) {
	w := newProgressWorld(t)
	got, err := w.cycleProgress(shared.Actor{UserID: uuid.NewV7()})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"Begin", "GetCycle", "Facts", "Authorize cycle.read", "StateGroups", "CycleItems", "CycleSnapshot", "Now", "End"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if len(w.viewers) != 1 || w.viewers[0] != nil {
		t.Fatalf("viewers %v, want every account's (nil)", w.viewers)
	}
	if want := (domain.CycleProgress{Status: domain.StatusCurrent, Live: progressWorldLive(t), Snapshot: &progressFrozen}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%+v\nwant\n%+v", got, want)
	}
	delete(w.snapshots, lockCycle)
	if got, err := w.cycleProgress(shared.Actor{UserID: uuid.NewV7()}); err != nil || got.Snapshot != nil {
		t.Fatalf("not frozen: %+v, %v; want no snapshot", got.Snapshot, err)
	}
}

// A caller who sees only his own work items (OwnOnly) counts his own alone, the issue module asked for his, and is
// not shown the frozen progress, of every account's work items, which is not even read (M6 P4 spec R5).
func TestGetCycleProgressOfAnOwnOnlyCaller(t *testing.T) {
	w := newProgressWorld(t)
	w.grant = shared.Grant{ProjectRole: shared.RoleGuest, OwnOnly: true}
	caller := shared.Actor{UserID: uuid.NewV7()}
	got, err := w.cycleProgress(caller)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "GetCycle", "Facts", "Authorize cycle.read", "StateGroups", "CycleItems", "Now", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if len(w.viewers) != 1 || w.viewers[0] == nil || *w.viewers[0] != caller.UserID {
		t.Fatalf("viewers %v, want the caller %s", w.viewers, caller.UserID)
	}
	if want := (domain.CycleProgress{Status: domain.StatusCurrent, Live: progressWorldLive(t)}); !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%+v\nwant\n%+v", got, want)
	}
}

// A module's progress is read as a cycle's, module.read, its burndown from its start to its target day; a module is
// never frozen. An OwnOnly caller's counts his own alone. A module without both days has no burndown.
func TestGetModuleProgress(t *testing.T) {
	w := newProgressWorld(t)
	got, err := w.moduleProgress(shared.Actor{UserID: uuid.NewV7()})
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "GetModule", "Facts", "Authorize module.read", "StateGroups", "ModuleItems", "Now", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if len(w.viewers) != 1 || w.viewers[0] != nil {
		t.Fatalf("viewers %v, want every account's (nil)", w.viewers)
	}
	if want := progressWorldLive(t); !reflect.DeepEqual(got, want) {
		t.Fatalf("got\n%+v\nwant\n%+v", got, want)
	}

	w = newProgressWorld(t)
	w.grant = shared.Grant{ProjectRole: shared.RoleGuest, OwnOnly: true}
	caller := shared.Actor{UserID: uuid.NewV7()}
	if _, err := w.moduleProgress(caller); err != nil || len(w.viewers) != 1 || w.viewers[0] == nil || *w.viewers[0] != caller.UserID {
		t.Fatalf("an OwnOnly caller: %v, viewers %v; want the caller %s", err, w.viewers, caller.UserID)
	}

	w = newProgressWorld(t)
	m := w.modules[lockModule]
	m.StartDate = nil
	w.modules[lockModule] = m
	if got, err := w.moduleProgress(shared.Actor{UserID: uuid.NewV7()}); err != nil || got.Burndown != nil || got.Total != 2 {
		t.Fatalf("no start: %+v, %v; want the counts without a burndown", got, err)
	}
}

// What a progress refuses is what a read refuses (TestReadsRefuse): what it names not there or not seen, its 404; a
// role refused, the 403; then nothing of the work items is read. An OwnOnly grant, an archived project, cycle or
// module, and cycles or modules turned off are read.
func TestProgressReadsRefuse(t *testing.T) {
	forbidden := shared.Forbidden()
	archived := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name                  string
		change                func(w *lockWorld)
		cycleWant, moduleWant error
	}{
		{"not there", func(w *lockWorld) { delete(w.cycles, lockCycle); delete(w.modules, lockModule) }, domain.ErrCycleNotFound, domain.ErrModuleNotFound},
		{"no project", func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrCycleNotFound, domain.ErrModuleNotFound},
		{"not seen", func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrCycleNotFound, domain.ErrModuleNotFound},
		{"a role refused", func(w *lockWorld) { w.refusal = forbidden }, forbidden, forbidden},
		{"own only", func(w *lockWorld) { w.grant.OwnOnly = true }, nil, nil},
		{"archived", func(w *lockWorld) {
			f := w.facts[lockProject]
			f.Archived, f.CycleView, f.ModuleView = true, false, false
			w.facts[lockProject] = f
			c, m := w.cycles[lockCycle], w.modules[lockModule]
			c.ArchivedAt, m.ArchivedAt = &archived, &archived
			w.cycles[lockCycle], w.modules[lockModule] = c, m
		}, nil, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newProgressWorld(t)
			tt.change(w)
			if _, err := w.cycleProgress(shared.Actor{UserID: uuid.NewV7()}); !sameRefusal(err, tt.cycleWant) || (tt.cycleWant == nil) != (err == nil) {
				t.Errorf("cycle = %v, want %v", err, tt.cycleWant)
			}
			w2 := newProgressWorld(t)
			tt.change(w2)
			if _, err := w2.moduleProgress(shared.Actor{UserID: uuid.NewV7()}); !sameRefusal(err, tt.moduleWant) || (tt.moduleWant == nil) != (err == nil) {
				t.Errorf("module = %v, want %v", err, tt.moduleWant)
			}
			for _, w := range []*lockWorld{w, w2} {
				if read := slices.Contains(w.calls, "StateGroups"); read != (tt.cycleWant == nil) {
					t.Errorf("calls %q: the work items read %t, want %t", w.calls, read, tt.cycleWant == nil)
				}
			}
		})
	}
}

// What a progress cannot be computed from is its failure, a defect rather than numbers: a work item in a state not
// the project's, a time zone that does not load.
func TestProgressPassesItsFailures(t *testing.T) {
	w := newProgressWorld(t)
	w.groups = map[uuid.UUID]domain.StateGroup{progressBacklog: domain.GroupBacklog}
	if _, err := w.cycleProgress(shared.Actor{UserID: uuid.NewV7()}); err == nil {
		t.Error("a work item in no state of the project's: no error")
	}
	w = newProgressWorld(t)
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Mars/Olympus", ModuleView: true}
	if _, err := w.moduleProgress(shared.Actor{UserID: uuid.NewV7()}); err == nil || errors.Is(err, domain.ErrModuleNotFound) {
		t.Errorf("a time zone that does not load: %v, want a failure", err)
	}
}
