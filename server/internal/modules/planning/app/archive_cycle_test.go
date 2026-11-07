package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (w *lockWorld) SetCycleArchived(ctx context.Context, c domain.Cycle) error {
	w.log(ctx, fmt.Sprintf("SetCycleArchived %t", c.ArchivedAt != nil))
	w.cycles[c.ID] = c
	w.updated = append(w.updated, c)
	return nil
}

func (w *lockWorld) archive(t *testing.T, archive bool) (domain.ReadCycle, error) {
	t.Helper()
	d := ArchiveCycleDeps{Locks: w.locks(), Cycles: w, Clock: w}
	uc := NewUnarchiveCycle(d)
	if archive {
		uc = NewArchiveCycle(d)
	}
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle)
}

// completedSprint makes the world's Sprint completed on its project's day: now is the 14th of November in Shanghai
// (17:00 UTC on the 13th), the day after Sprint's last.
func completedSprint(w *lockWorld) {
	w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "Asia/Shanghai", CycleView: true}
	w.now = time.Date(2026, 11, 13, 17, 0, 0, 0, time.UTC)
}

// Archiving takes the lock path on the cycle; under the lock, the clock, then the write: archived at that moment,
// last changed by the caller at the same moment. The answer is the cycle as stored.
func TestArchiveCycle(t *testing.T) {
	w := sprintWorld(t)
	completedSprint(w)
	got, err := w.archive(t, true)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
		"GetCycle", "Authorize cycle.write", "Now", "SetCycleArchived true", "End", "Leave"}
	if !slices.Equal(w.calls, want) {
		t.Fatalf("calls\n%q\nwant\n%q", w.calls, want)
	}
	stored := w.updated[0]
	if stored.ArchivedAt == nil || !stored.ArchivedAt.Equal(w.now) || !stored.UpdatedAt.Equal(w.now) || *stored.UpdatedByID != lockEditor ||
		stored.Name != "Sprint" || *stored.OwnedByID != lockCreator {
		t.Fatalf("stored %+v", stored)
	}
	if got.ArchivedAt == nil || got.Status != domain.StatusCompleted {
		t.Fatalf("answer %+v", got)
	}
}

// Only a completed cycle is archived (planning.cycle_not_completed): not a current one, nor one that is current on
// its project's day while completed in UTC (Los Angeles, 05:00 UTC on the 14th), nor a draft. An archived cycle
// archived again, and one not archived unarchived, answer as read and write nothing (M6 P1 spec R3).
func TestArchiveCycleRefusesAndRepeats(t *testing.T) {
	losAngeles := func(w *lockWorld) {
		w.facts[lockProject] = ProjectFacts{WorkspaceID: lockWorkspace, Timezone: "America/Los_Angeles", CycleView: true}
		w.now = time.Date(2026, 11, 14, 5, 0, 0, 0, time.UTC)
	}
	draft := func(w *lockWorld) { c := w.cycles[lockCycle]; c.Dates = domain.Dates{}; w.cycles[lockCycle] = c }
	for name, change := range map[string]func(w *lockWorld){"current": nil, "current in Los Angeles": losAngeles, "a draft": draft} {
		w := sprintWorld(t)
		if change != nil {
			change(w)
		}
		if _, err := w.archive(t, true); !errors.Is(err, domain.ErrCycleNotCompleted) || len(w.updated) != 0 {
			t.Errorf("%s: %v, stored %v; want planning.cycle_not_completed", name, err, w.updated)
		}
	}
	archived := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	for _, archive := range []bool{true, false} {
		w := sprintWorld(t)
		completedSprint(w)
		if archive {
			c := w.cycles[lockCycle]
			c.ArchivedAt = &archived
			w.cycles[lockCycle] = c
		}
		got, err := w.archive(t, archive)
		if err != nil || len(w.updated) != 0 || (got.ArchivedAt != nil) != archive || got.Status != domain.StatusCompleted {
			t.Errorf("archive=%t again: %+v, %v, stored %v", archive, got, err, w.updated)
		}
	}
}

// Unarchiving clears the archive, whatever the cycle's status, last changed by the caller at the moment read after
// the lock.
func TestUnarchiveCycle(t *testing.T) {
	w := sprintWorld(t)
	archived := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
	c := w.cycles[lockCycle]
	c.ArchivedAt = &archived
	w.cycles[lockCycle] = c
	got, err := w.archive(t, false)
	if err != nil {
		t.Fatal(err)
	}
	if stored := w.updated[0]; stored.ArchivedAt != nil || !stored.UpdatedAt.Equal(w.now) || *stored.UpdatedByID != lockEditor ||
		got.ArchivedAt != nil || got.Status != domain.StatusCurrent || !slices.Contains(w.calls, "SetCycleArchived false") {
		t.Fatalf("stored %+v, answer %+v, calls %q", stored, got, w.calls)
	}
}
