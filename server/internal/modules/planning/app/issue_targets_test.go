package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// targetWorld is IssueTargets' ports as fakes that log their calls: the project's facts, the cycles GetCycle reads, and
// what the targets' and the open ids' reads answer, the store keeping to the project it is asked about.
type targetWorld struct {
	calls                       []string
	facts                       map[uuid.UUID]ProjectFacts
	cycles                      map[uuid.UUID]domain.Cycle
	cycleTargets, moduleTargets []IssueTarget
	openCycles, openModules     []uuid.UUID
	fail                        error // every read's failure
}

func (w *targetWorld) Facts(_ context.Context, id uuid.UUID) (ProjectFacts, bool, error) {
	w.calls = append(w.calls, "Facts")
	f, ok := w.facts[id]
	return f, ok, w.fail
}

func (w *targetWorld) GetCycle(_ context.Context, id uuid.UUID) (domain.Cycle, bool, error) {
	w.calls = append(w.calls, "GetCycle")
	c, ok := w.cycles[id]
	return c, ok, w.fail
}

func (w *targetWorld) CycleTargets(_ context.Context, project uuid.UUID, ids []uuid.UUID) ([]IssueTarget, error) {
	w.calls = append(w.calls, fmt.Sprint("CycleTargets ", project == targetProject, " ", len(ids)))
	return w.cycleTargets, w.fail
}

func (w *targetWorld) ModuleTargets(_ context.Context, project uuid.UUID, ids []uuid.UUID) ([]IssueTarget, error) {
	w.calls = append(w.calls, fmt.Sprint("ModuleTargets ", project == targetProject, " ", len(ids)))
	return w.moduleTargets, w.fail
}

func (w *targetWorld) OpenCycleIDs(_ context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error) {
	w.calls = append(w.calls, fmt.Sprint("OpenCycleIDs ", project == targetProject, " ", today))
	return w.openCycles, w.fail
}

func (w *targetWorld) OpenModuleIDs(_ context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error) {
	w.calls = append(w.calls, fmt.Sprint("OpenModuleIDs ", project == targetProject, " ", today))
	return w.openModules, w.fail
}

var (
	targetProject = uuid.NewV7()
	targetCycle   = uuid.NewV7()
	// 2026-11-07 17:00 UTC is the 8th in Shanghai, still the 7th in UTC (and noon in New York); 2026-11-08 03:00 UTC is
	// the 8th in UTC, still the 7th in New York (22:00).
	shanghaiNextDay = time.Date(2026, 11, 7, 17, 0, 0, 0, time.UTC)
	newYorkDayLeft  = time.Date(2026, 11, 8, 3, 0, 0, 0, time.UTC)
)

func newTargetWorld() *targetWorld {
	return &targetWorld{
		facts:  map[uuid.UUID]ProjectFacts{targetProject: {Timezone: "Asia/Shanghai", CycleView: true, ModuleView: true}},
		cycles: map[uuid.UUID]domain.Cycle{targetCycle: {ID: targetCycle, ProjectID: targetProject, Name: "Sprint"}},
	}
}

func (w *targetWorld) issueTargets() *IssueTargets { return NewIssueTargets(w, w) }

// Targets reads the project's facts, then, of the cycles and of the modules asked about, the project's undeleted ones,
// archived or not, as the store reads them: the toggles are the project's, each its own, the targets as read.
func TestIssueTargetsAnswerTheTogglesAndTheTargets(t *testing.T) {
	c1, c2, m1 := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	for _, on := range []struct{ cycles, modules bool }{{true, false}, {false, true}} {
		w := newTargetWorld()
		w.facts[targetProject] = ProjectFacts{Timezone: "UTC", CycleView: on.cycles, ModuleView: on.modules}
		w.cycleTargets = []IssueTarget{{ID: c1, Name: "Sprint 1"}, {ID: c2, Name: "Sprint 2", Archived: true}}
		w.moduleTargets = []IssueTarget{{ID: m1, Name: "Core"}}
		got, err := w.issueTargets().Targets(context.Background(), targetProject, []uuid.UUID{c1, c2}, []uuid.UUID{m1})
		if err != nil {
			t.Fatal(err)
		}
		want := IssueTargetFacts{CyclesEnabled: on.cycles, ModulesEnabled: on.modules, Cycles: w.cycleTargets, Modules: w.moduleTargets}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Targets() = %+v\nwant %+v", got, want)
		}
		if want := []string{"Facts", "CycleTargets true 2", "ModuleTargets true 1"}; !slices.Equal(w.calls, want) {
			t.Errorf("calls %q, want %q", w.calls, want)
		}
	}
}

// Of the targets' reads, only those of what is asked about are made: none asked about, the toggles alone.
func TestIssueTargetsReadOnlyWhatIsAskedAbout(t *testing.T) {
	w := newTargetWorld()
	got, err := w.issueTargets().Targets(context.Background(), targetProject, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if want := (IssueTargetFacts{CyclesEnabled: true, ModulesEnabled: true}); !reflect.DeepEqual(got, want) {
		t.Errorf("Targets() = %+v, want the toggles alone", got)
	}
	if want := []string{"Facts"}; !slices.Equal(w.calls, want) {
		t.Errorf("calls %q, want %q", w.calls, want)
	}
	w.calls = nil
	if _, err := w.issueTargets().Targets(context.Background(), targetProject, nil, []uuid.UUID{uuid.NewV7()}); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Facts", "ModuleTargets true 1"}; !slices.Equal(w.calls, want) {
		t.Errorf("calls %q, want %q", w.calls, want)
	}
}

// A cycle has ended when its status on the day now falls on in its project's time zone is completed: after its end
// day there, not on it; one without dates, not begun or in progress has not. The cycle is read, then the project's
// facts.
func TestCycleCompletedOnTheProjectsDay(t *testing.T) {
	for _, c := range []struct {
		name, zone, start, end string
		now                    time.Time
		want                   bool
	}{
		{"ended the day before in Shanghai, not yet in UTC", "Asia/Shanghai", "2026-11-01", "2026-11-07", shanghaiNextDay, true},
		{"ending on the day in Shanghai", "Asia/Shanghai", "2026-11-01", "2026-11-08", shanghaiNextDay, false},
		{"ended the day before in UTC", "UTC", "2026-11-01", "2026-11-07", newYorkDayLeft, true},
		{"ended the day before in UTC, not yet in New York", "America/New_York", "2026-11-01", "2026-11-07", newYorkDayLeft, false},
		{"draft", "Asia/Shanghai", "", "", shanghaiNextDay, false},
		{"upcoming", "Asia/Shanghai", "2026-11-09", "2026-11-10", shanghaiNextDay, false},
		{"current", "Asia/Shanghai", "2026-11-01", "2026-11-10", shanghaiNextDay, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			w := newTargetWorld()
			w.facts[targetProject] = ProjectFacts{Timezone: c.zone}
			cy := w.cycles[targetCycle]
			if c.start != "" {
				cy.Dates = domain.Dates{Start: date(t, c.start), End: date(t, c.end)}
			}
			w.cycles[targetCycle] = cy
			got, err := w.issueTargets().CycleCompleted(context.Background(), targetProject, targetCycle, c.now)
			if err != nil || got != c.want {
				t.Fatalf("CycleCompleted() = %v, %v; want %v", got, err, c.want)
			}
			if want := []string{"GetCycle", "Facts"}; !slices.Equal(w.calls, want) {
				t.Errorf("calls %q, want %q", w.calls, want)
			}
		})
	}
}

// The open ids are the store's, asked for on the day now falls on in the project's time zone.
func TestOpenTargetsOnTheProjectsDay(t *testing.T) {
	for _, c := range []struct {
		zone, day string
		now       time.Time
	}{
		{"Asia/Shanghai", "2026-11-08", shanghaiNextDay},
		{"UTC", "2026-11-07", shanghaiNextDay},
		{"America/New_York", "2026-11-07", newYorkDayLeft},
	} {
		w := newTargetWorld()
		w.facts[targetProject] = ProjectFacts{Timezone: c.zone}
		w.openCycles, w.openModules = []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, []uuid.UUID{uuid.NewV7()}
		cycles, modules, err := w.issueTargets().OpenTargets(context.Background(), targetProject, c.now)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(cycles, w.openCycles) || !slices.Equal(modules, w.openModules) {
			t.Errorf("%s: OpenTargets() = %v, %v; want %v, %v", c.zone, cycles, modules, w.openCycles, w.openModules)
		}
		if want := []string{"Facts", "OpenCycleIDs true " + c.day, "OpenModuleIDs true " + c.day}; !slices.Equal(w.calls, want) {
			t.Errorf("%s: calls %q, want %q", c.zone, w.calls, want)
		}
	}
}

// The issue write that asks holds its project's lock, and Targets answered it the cycle it asks about: a project that
// is not there, a cycle that is not the project's undeleted one and a time zone that does not load are internal
// errors, and a read's failure is the answer, nothing read after it.
func TestIssueTargetsRefuseWhatTheLockedWriteCannotAsk(t *testing.T) {
	ctx := context.Background()
	gone, goneCycle := uuid.NewV7(), uuid.NewV7()
	w := newTargetWorld()
	w.cycles[goneCycle] = domain.Cycle{ID: goneCycle, ProjectID: gone}
	if _, err := w.issueTargets().Targets(ctx, gone, []uuid.UUID{goneCycle}, nil); err == nil {
		t.Error("Targets() of a project that is not there: no error")
	}
	if _, err := w.issueTargets().CycleCompleted(ctx, gone, goneCycle, shanghaiNextDay); err == nil {
		t.Error("CycleCompleted() of a project that is not there: no error")
	}
	if _, _, err := w.issueTargets().OpenTargets(ctx, gone, shanghaiNextDay); err == nil {
		t.Error("OpenTargets() of a project that is not there: no error")
	}
	if want := []string{"Facts", "GetCycle", "Facts", "Facts"}; !slices.Equal(w.calls, want) {
		t.Errorf("calls %q, want %q: nothing read after the missing project", w.calls, want)
	}

	other := newTargetWorld()
	other.cycles[targetCycle] = domain.Cycle{ID: targetCycle, ProjectID: uuid.NewV7()}
	if _, err := other.issueTargets().CycleCompleted(ctx, targetProject, targetCycle, shanghaiNextDay); err == nil {
		t.Error("CycleCompleted() of another project's cycle: no error")
	}
	if _, err := other.issueTargets().CycleCompleted(ctx, targetProject, uuid.NewV7(), shanghaiNextDay); err == nil {
		t.Error("CycleCompleted() of a cycle that is not there: no error")
	}

	bad := newTargetWorld()
	bad.facts[targetProject] = ProjectFacts{Timezone: "Mars/Olympus"}
	if _, err := bad.issueTargets().CycleCompleted(ctx, targetProject, targetCycle, shanghaiNextDay); err == nil {
		t.Error("CycleCompleted() in a time zone that does not load: no error")
	}
	if _, _, err := bad.issueTargets().OpenTargets(ctx, targetProject, shanghaiNextDay); err == nil {
		t.Error("OpenTargets() in a time zone that does not load: no error")
	}

	boom := errors.New("boom")
	failing := newTargetWorld()
	failing.fail = boom
	if _, err := failing.issueTargets().Targets(ctx, targetProject, []uuid.UUID{targetCycle}, nil); !errors.Is(err, boom) {
		t.Errorf("Targets() = %v; want the read's failure", err)
	}
	if _, err := failing.issueTargets().CycleCompleted(ctx, targetProject, targetCycle, shanghaiNextDay); !errors.Is(err, boom) {
		t.Errorf("CycleCompleted() = %v; want the read's failure", err)
	}
	if _, _, err := failing.issueTargets().OpenTargets(ctx, targetProject, shanghaiNextDay); !errors.Is(err, boom) {
		t.Errorf("OpenTargets() = %v; want the read's failure", err)
	}
	if want := []string{"Facts", "GetCycle", "Facts"}; !slices.Equal(failing.calls, want) {
		t.Errorf("calls %q, want %q: nothing read after a failure", failing.calls, want)
	}
}
