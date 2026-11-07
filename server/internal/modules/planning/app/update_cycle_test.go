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

func (w *lockWorld) UpdateCycle(ctx context.Context, c domain.Cycle) error {
	w.log(ctx, "UpdateCycle")
	w.cycles[c.ID] = c
	w.updated = append(w.updated, c)
	return nil
}

var lockEditor = uuid.NewV7()

func (w *lockWorld) update(t *testing.T, p domain.CyclePatch) (domain.ReadCycle, error) {
	t.Helper()
	uc := NewUpdateCycle(UpdateCycleDeps{Locks: w.locks(), Cycles: w, Clock: w})
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle, p)
}

func str(s string) *string { return &s }

// sprintWorld is a world whose cycle is Sprint, the 2nd to the 13th of November 2026, made by its creator an hour
// before; now is the 7th, noon, in UTC: Sprint is current.
func sprintWorld(t *testing.T) *lockWorld {
	w := newLockWorld()
	made := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
	w.cycles[lockCycle] = domain.Cycle{ID: lockCycle, ProjectID: lockProject, WorkspaceID: lockWorkspace, Name: "Sprint", SortOrder: 5,
		Dates: domain.Dates{Start: date(t, "2026-11-02"), End: date(t, "2026-11-13")}, OwnedByID: &lockCreator, CreatedAt: made, UpdatedAt: made,
		CreatedByID: &lockCreator, UpdatedByID: &lockCreator}
	w.now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	return w
}

// A change takes the lock path on the cycle (Locks); under the lock, the clock, then the other cycles' dates when its
// dates change, then the write: the fields it gives, last changed by the caller at that moment, its owner and creation
// kept. The answer is the cycle as stored, with its status.
func TestUpdateCycle(t *testing.T) {
	w := sprintWorld(t)
	got, err := w.update(t, domain.CyclePatch{Name: str(" Sprint 2 "), SetEnd: true, End: date(t, "2026-11-20")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
		"GetCycle", "Authorize cycle.write", "Now", "ProjectCycleDates", "UpdateCycle", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	stored := w.updated[0]
	old := sprintWorld(t).cycles[lockCycle]
	if stored.Name != "Sprint 2" || stored.Dates.End.String() != "2026-11-20" || stored.Dates.Start.String() != "2026-11-02" ||
		!stored.UpdatedAt.Equal(w.now) || *stored.UpdatedByID != lockEditor || *stored.OwnedByID != lockCreator ||
		*stored.CreatedByID != lockCreator || !stored.CreatedAt.Equal(old.CreatedAt) || stored.SortOrder != 5 {
		t.Fatalf("stored %+v", stored)
	}
	if got.Name != "Sprint 2" || got.Status != domain.StatusCurrent {
		t.Fatalf("answer %+v", got)
	}
	// Without new dates — none given, or its own given again — no other cycle is read.
	for _, p := range []domain.CyclePatch{{SortOrder: ptrTo(-1.0)}, {Name: str("Two"), SetStart: true, Start: date(t, "2026-11-02"), SetEnd: true,
		End: date(t, "2026-11-13")}} {
		w = sprintWorld(t)
		if _, err := w.update(t, p); err != nil || slices.Contains(w.calls, "ProjectCycleDates") || len(w.updated) != 1 {
			t.Fatalf("%+v: %q, %v", p, w.calls, err)
		}
	}
}

func ptrTo[T any](v T) *T { return &v }

// A change of nothing — none given, or each as it is — writes nothing, and its answer is the cycle as read.
func TestUpdateCycleChangesNothing(t *testing.T) {
	for _, p := range []domain.CyclePatch{{}, {Name: str("Sprint"), SortOrder: ptrTo(5.0), SetStart: true, Start: nil}} {
		w := sprintWorld(t)
		if p.SetStart {
			p.Start = date(t, "2026-11-02")
		}
		got, err := w.update(t, p)
		if err != nil || len(w.updated) != 0 || !got.UpdatedAt.Equal(w.cycles[lockCycle].UpdatedAt) || got.Status != domain.StatusCurrent {
			t.Fatalf("%+v: %+v, %v, stored %v", p, got, err, w.updated)
		}
	}
}

// The overlap check leaves the cycle itself out: Sprint moved onto days it had is in order; onto another cycle's,
// refused, nothing written.
func TestUpdateCycleOverlap(t *testing.T) {
	w := sprintWorld(t)
	if _, err := w.update(t, domain.CyclePatch{SetStart: true, Start: date(t, "2026-11-05"), SetEnd: true, End: date(t, "2026-11-10")}); err != nil {
		t.Fatalf("within its own days: %v", err)
	}
	w = sprintWorld(t)
	other := uuid.NewV7()
	w.cycles[other] = domain.Cycle{ID: other, ProjectID: lockProject, Dates: domain.Dates{Start: date(t, "2026-11-14"), End: date(t, "2026-11-20")}}
	_, err := w.update(t, domain.CyclePatch{SetEnd: true, End: date(t, "2026-11-14")})
	if !errors.Is(err, domain.ErrDatesOverlap) || len(w.updated) != 0 {
		t.Fatalf("onto another's day: %v, stored %v", err, w.updated)
	}
}

// The refusals in their order (M6 P1 spec R2, R5): a value's problem before anything is read; an archived cycle, any
// change, none too; a completed one, any field but the sort order, even as it is; then the merged dates (R4). A
// completed cycle takes its sort order. Completed is on the project's day: Sprint ends the 13th; at 17:00 UTC on the
// 13th it is the 14th in Shanghai, completed there and current in UTC; at 05:00 UTC on the 14th it is still the 13th in
// Los Angeles, current there and completed in UTC.
func TestUpdateCycleRefusals(t *testing.T) {
	shanghai := func(w *lockWorld) {
		w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true}
		w.now = time.Date(2026, 11, 13, 17, 0, 0, 0, time.UTC)
	}
	losAngeles := func(w *lockWorld) {
		w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "America/Los_Angeles", CycleView: true}
		w.now = time.Date(2026, 11, 14, 5, 0, 0, 0, time.UTC)
	}
	archived := func(w *lockWorld) {
		c := w.cycles[lockCycle]
		at := time.Date(2026, 11, 2, 0, 0, 0, 0, time.UTC)
		c.ArchivedAt = &at
		w.cycles[lockCycle] = c
	}
	for _, tt := range []struct {
		name   string
		change func(w *lockWorld)
		p      domain.CyclePatch
		want   string // the error's code, "" for none
	}{
		{"an empty name, completed", shanghai, domain.CyclePatch{Name: str(" ")}, "validation_failed"},
		{"archived, nothing given", archived, domain.CyclePatch{}, "planning.archived"},
		{"archived and completed", func(w *lockWorld) { archived(w); shanghai(w) }, domain.CyclePatch{Name: str("x")}, "planning.archived"},
		{"completed in Shanghai, a name", shanghai, domain.CyclePatch{Name: str("x")}, "planning.cycle_completed"},
		{"completed, its own name", shanghai, domain.CyclePatch{Name: str("Sprint")}, "planning.cycle_completed"},
		{"completed, its sort order", shanghai, domain.CyclePatch{SortOrder: ptrTo(1.0)}, ""},
		{"completed, nothing given", shanghai, domain.CyclePatch{}, ""},
		{"current in Los Angeles, a name", losAngeles, domain.CyclePatch{Name: str("x")}, ""},
		{"the start alone after the end", nil, domain.CyclePatch{SetStart: true, Start: date(t, "2026-11-14")}, "validation_failed"},
	} {
		w := sprintWorld(t)
		if tt.change != nil {
			tt.change(w)
		}
		_, err := w.update(t, tt.p)
		var e *shared.Error
		switch {
		case tt.want == "" && err != nil:
			t.Errorf("%s: %v, want none", tt.name, err)
		case tt.want != "" && (!errors.As(err, &e) || e.Code != tt.want):
			t.Errorf("%s: %v, want %s", tt.name, err, tt.want)
		case tt.want != "" && len(w.updated) != 0:
			t.Errorf("%s: stored %v", tt.name, w.updated)
		}
		if tt.name == "an empty name, completed" && len(w.calls) != 0 {
			t.Errorf("a value's problem read %q", w.calls)
		}
	}
}
