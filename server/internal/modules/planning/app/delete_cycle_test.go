package app

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (w *lockWorld) DeleteCycle(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, fmt.Sprintf("DeleteCycle %t %t %t", id == lockCycle, at.Equal(w.now), by == lockEditor))
	delete(w.cycles, id)
	return nil
}

// Deleting takes the lock path on the cycle, deciding cycle.delete, the cycle's creator in the target (AllowCreator);
// under the lock, the clock, then the deletion: deleted and last changed by the caller at that moment. Neither its
// status nor its archive holds it back (M6 P1 spec R5).
func TestDeleteCycle(t *testing.T) {
	archived := func(w *lockWorld) {
		c := w.cycles[lockCycle]
		at := time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)
		c.ArchivedAt = &at
		w.cycles[lockCycle] = c
		completedSprint(w)
	}
	for name, change := range map[string]func(*lockWorld){"current": nil, "completed and archived": archived} {
		w := sprintWorld(t)
		if change != nil {
			change(w)
		}
		uc := NewDeleteCycle(DeleteCycleDeps{Locks: w.locks(), Cycles: w, Clock: w})
		if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.delete outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetCycle", "Authorize cycle.delete", "Now", "DeleteCycle true true true", "End", "Leave"}
		if !slices.Equal(w.calls, want) {
			t.Fatalf("%s: calls\n%q\nwant\n%q", name, w.calls, want)
		}
		for _, target := range w.targets {
			if target.CreatorID != lockCreator {
				t.Fatalf("%s: target %+v, want its creator", name, target)
			}
		}
	}
}
