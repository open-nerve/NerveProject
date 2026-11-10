package app

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListCycles is the lock world's list: the project's cycles, archived or not as asked, in the map's id order (a
// store orders them; the use case keeps its order).
func (w *lockWorld) ListCycles(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Cycle, error) {
	if archived {
		w.log(ctx, "ListCycles archived")
	} else {
		w.log(ctx, "ListCycles")
	}
	var out []domain.Cycle
	for _, c := range w.cycles {
		if c.ProjectID == project && (c.ArchivedAt != nil) == archived {
			out = append(out, c)
		}
	}
	slices.SortFunc(out, func(a, b domain.Cycle) int { return a.ID.Compare(b.ID) })
	return out, nil
}

func (w *lockWorld) reads() Reads { return NewReads(w, w, w, w) }

func (w *lockWorld) get(id uuid.UUID) (domain.ReadCycle, error) {
	uc := NewGetCycle(GetCycleDeps{Reads: w.reads(), Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: uuid.NewV7()}), id)
}

func (w *lockWorld) list(project uuid.UUID, archived bool) ([]domain.CycleListItem, error) {
	return w.listAs(shared.Actor{UserID: uuid.NewV7()}, project, archived)
}

func (w *lockWorld) listAs(actor shared.Actor, project uuid.UUID, archived bool) ([]domain.CycleListItem, error) {
	uc := NewListCycles(ListCyclesDeps{Reads: w.reads(), Cycles: w, Groups: w, Stats: w, Snapshots: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), actor), project, archived)
}

// A read decides in one transaction of its own, the read-only snapshot, where it reads the cycle, then its project's
// facts, then authorizes cycle.read on them, the cycle's creator in the target; no gate and no lock. The clock is
// read last, once, and the status is the cycle's on that moment's day in the project's time zone.
func TestGetCycle(t *testing.T) {
	w := newLockWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true}
	c := w.cycles[lockCycle]
	c.Dates = domain.Dates{Start: date(t, "2026-11-08"), End: date(t, "2026-11-14")}
	w.cycles[lockCycle] = c
	// 2026-11-07 17:00 UTC is the 8th in Shanghai: current there, upcoming in UTC.
	w.now = time.Date(2026, 11, 7, 17, 0, 0, 0, time.UTC)
	got, err := w.get(lockCycle)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Begin", "GetCycle", "Facts", "Authorize cycle.read", "Now", "End"}; !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	if got.ID != lockCycle || got.Status != domain.StatusCurrent {
		t.Fatalf("got %+v, want the cycle, current", got)
	}
	if want := (shared.Target{WorkspaceID: lockWorkspace, ProjectID: lockProject, CreatorID: lockCreator}); !slices.Equal(w.targets, []shared.Target{want}) {
		t.Fatalf("targets %+v", w.targets)
	}
}

// The project's cycles, archived or not as asked, in the store's order, each with its status on the project's day at
// the clock's one reading, after the list.
func TestListCycles(t *testing.T) {
	w := newLockWorld()
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true}
	c := w.cycles[lockCycle]
	c.Dates = domain.Dates{Start: date(t, "2026-11-07"), End: date(t, "2026-11-07")}
	w.cycles[lockCycle] = c
	archived := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
	other := uuid.NewV7()
	w.cycles[other] = domain.Cycle{ID: other, ProjectID: lockProject, Dates: domain.Dates{Start: date(t, "2026-10-01"), End: date(t, "2026-10-14")},
		ArchivedAt: &archived}
	elsewhere := uuid.NewV7()
	w.cycles[elsewhere] = domain.Cycle{ID: elsewhere, ProjectID: uuid.NewV7()}
	// 2026-11-06 17:00 UTC is the 7th in Shanghai: the cycle of the 7th is current there, upcoming in UTC.
	w.now = time.Date(2026, 11, 6, 17, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		archived bool
		list     string
		want     []uuid.UUID
		status   domain.Status
	}{{false, "ListCycles", []uuid.UUID{lockCycle}, domain.StatusCurrent}, {true, "ListCycles archived", []uuid.UUID{other}, domain.StatusCompleted}} {
		w.calls = nil
		got, err := w.list(lockProject, tt.archived)
		if err != nil {
			t.Fatal(err)
		}
		if want := []string{"Begin", "Facts", "Authorize cycle.read", tt.list, "StateGroups", "CycleCounts", "CycleAssignees", "CycleSnapshotCounts",
			"Now", "End"}; !slices.Equal(w.calls, want) {
			t.Fatalf("archived=%t: calls\n%q\nwant\n%q", tt.archived, w.calls, want)
		}
		var ids []uuid.UUID
		for _, c := range got {
			ids = append(ids, c.ID)
			if c.Status != tt.status {
				t.Errorf("archived=%t: %s is %s, want %s", tt.archived, c.ID, c.Status, tt.status)
			}
		}
		if !slices.Equal(ids, tt.want) {
			t.Errorf("archived=%t: cycles %v, want %v", tt.archived, ids, tt.want)
		}
	}
	// A project without cycles lists none, as an empty list.
	empty := uuid.NewV7()
	w.facts[empty] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "UTC"}
	if got, err := w.list(empty, false); err != nil || got == nil || len(got) != 0 {
		t.Fatalf("an empty project: %v, %v; want [], nil", got, err)
	}
}

// What a read refuses: what it names not there, or not seen, is its 404 — planning.cycle_not_found for a cycle,
// whose project is gone too, project.not_found for a project; a role the rule refuses, the Authorizer's 403. What a
// write refuses and a read does not (M6 P1 spec R6, R7): an OwnOnly grant, an archived project, cycles turned off.
func TestReadsRefuse(t *testing.T) {
	forbidden := shared.Forbidden()
	for _, tt := range []struct {
		name              string
		change            func(w *lockWorld)
		getWant, listWant error
	}{
		{"no cycle", func(w *lockWorld) { delete(w.cycles, lockCycle) }, domain.ErrCycleNotFound, nil},
		{"no project", func(w *lockWorld) { delete(w.facts, lockProject) }, domain.ErrCycleNotFound, domain.ErrProjectNotFound},
		{"not seen", func(w *lockWorld) { w.refusal = shared.ErrNotVisible }, domain.ErrCycleNotFound, domain.ErrProjectNotFound},
		{"a role refused", func(w *lockWorld) { w.refusal = forbidden }, forbidden, forbidden},
		{"own only", func(w *lockWorld) { w.grant.OwnOnly = true }, nil, nil},
		{"archived project", func(w *lockWorld) { f := w.facts[lockProject]; f.Archived = true; w.facts[lockProject] = f }, nil, nil},
		{"cycles off", func(w *lockWorld) { f := w.facts[lockProject]; f.CycleView = false; w.facts[lockProject] = f }, nil, nil},
	} {
		w := newLockWorld()
		tt.change(w)
		if _, err := w.get(lockCycle); !errors.Is(err, tt.getWant) || (tt.getWant == nil) != (err == nil) {
			t.Errorf("%s: get = %v, want %v", tt.name, err, tt.getWant)
		}
		w = newLockWorld()
		tt.change(w)
		if _, err := w.list(lockProject, false); !errors.Is(err, tt.listWant) || (tt.listWant == nil) != (err == nil) {
			t.Errorf("%s: list = %v, want %v", tt.name, err, tt.listWant)
		}
	}
}
