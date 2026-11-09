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

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (w *lockWorld) DeleteCycle(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, fmt.Sprintf("DeleteCycle %t %t %t", id == lockCycle, at.Equal(w.now), by == lockEditor))
	delete(w.cycles, id)
	return nil
}

func (w *lockWorld) DeleteCyclePreferenceRows(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, fmt.Sprintf("DeleteCyclePreferenceRows %t %t %t", cycle == lockCycle, at.Equal(w.now), by == lockEditor))
	return nil
}

// Deleting takes the lock path on the cycle, deciding cycle.delete, the cycle's creator in the target (AllowCreator);
// under the lock, the clock, then the deletion: the cycle, then its page's display settings' rows, each deleted and
// last changed by the caller at that moment, in the write's transaction (M6 P3b spec 2.5). Neither its status nor its
// archive holds it back (M6 P1 spec R5).
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
		uc := NewDeleteCycle(DeleteCycleDeps{Locks: w.locks(), Cycles: w, Clock: w, Events: &deletionEvents{lockWorld: w}})
		if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := []string{"GetCycle outside tx", "Facts outside tx", "Authorize cycle.delete outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetCycle", "Authorize cycle.delete", "Now", "DeleteCycle true true true", "DeleteCyclePreferenceRows true true true", "Publish planning.cycle_deleted",
			"End", "Leave"}
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

// deletionEvents is the lock world's publisher: each event it is given, logged as "Publish <type>" inside or outside
// the write's transaction, and its answer, refusal.
type deletionEvents struct {
	*lockWorld
	published []shared.Event
	refusal   error
}

func (p *deletionEvents) Publish(ctx context.Context, e shared.Event) error {
	p.log(ctx, "Publish "+e.Type)
	p.published = append(p.published, e)
	return p.refusal
}

// wantDeletion checks that events published one event, the deletion of kind, of the entity named name of the lock
// world's project, by the editor at the world's moment, the moment of the row's deletion (M6 P3a spec 2.5, R6).
func wantDeletion(t *testing.T, events *deletionEvents, kind string, entity shared.EntityRef, name string) {
	t.Helper()
	if len(events.published) != 1 {
		t.Fatalf("published %+v, want the one deletion", events.published)
	}
	got := events.published[0]
	editor := lockEditor
	want := shared.Event{ID: got.ID, Type: kind, OccurredAt: events.now, WorkspaceID: lockWorkspace, ProjectID: lockProject, ActorID: &editor,
		Entity: entity, Changes: []shared.Change{{Field: "name", Old: &name}}}
	if got.ID == uuid.Nil() || !reflect.DeepEqual(got, want) {
		t.Errorf("published %+v, want %+v", got, want)
	}
}

// Deleting a cycle publishes its deletion in the write's transaction, after its rows' (M6 P3a spec 2.5, R6): the
// cycle, its workspace and project, the caller at the moment of the deletion, and its name before it as the one
// change. A publication that fails fails the deletion: the transaction ends with its error, which rolls it back.
func TestDeleteCyclePublishesItsDeletion(t *testing.T) {
	w := sprintWorld(t)
	events := &deletionEvents{lockWorld: w}
	uc := NewDeleteCycle(DeleteCycleDeps{Locks: w.locks(), Cycles: w, Clock: w, Events: events})
	if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle); err != nil {
		t.Fatal(err)
	}
	wantDeletion(t, events, "planning.cycle_deleted", shared.EntityRef{Kind: "cycle", ID: lockCycle}, "Sprint")

	w = sprintWorld(t)
	events = &deletionEvents{lockWorld: w, refusal: errors.New("a subscriber failed")}
	uc = NewDeleteCycle(DeleteCycleDeps{Locks: w.locks(), Cycles: w, Clock: w, Events: events})
	if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockCycle); !errors.Is(err, events.refusal) {
		t.Fatalf("Execute() = %v, want the publication's error", err)
	}
	if want := []string{"DeleteCyclePreferenceRows true true true", "Publish planning.cycle_deleted", "End", "Leave"}; !slices.Equal(w.calls[len(w.calls)-4:], want) {
		t.Errorf("calls %q, want them to end %q", w.calls, want)
	}
}
