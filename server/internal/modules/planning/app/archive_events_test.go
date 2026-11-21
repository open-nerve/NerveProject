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

// wantArchive checks that published is one event, the archive of kind, of entity of the lock world's project, by the
// editor at at, the moment written into the entity's archived_at; with no changes (M7 P3c spec 2.6, R3).
func wantArchive(t *testing.T, published []shared.Event, kind string, entity shared.EntityRef, at time.Time) {
	t.Helper()
	if len(published) != 1 {
		t.Fatalf("published %+v, want the one archive", published)
	}
	got := published[0]
	editor := lockEditor
	want := shared.Event{ID: got.ID, Type: kind, OccurredAt: got.OccurredAt, WorkspaceID: lockWorkspace, ProjectID: lockProject, ActorID: &editor,
		Entity: entity}
	if got.ID == uuid.Nil() || !got.OccurredAt.Equal(at) || !reflect.DeepEqual(got, want) {
		t.Errorf("published %+v, want %+v at %v", got, want, at)
	}
}

// ticking is the lock world's clock a second later after each read: a use case that read it again for its event
// would publish another moment than the one it wrote.
type ticking struct{ *lockWorld }

func (c ticking) Now() time.Time {
	now := c.lockWorld.Now()
	c.now = c.now.Add(time.Second)
	return now
}

// clockReads counts the clock's reads among calls.
func clockReads(calls []string) int {
	n := 0
	for _, call := range calls {
		if call == "Now" {
			n++
		}
	}
	return n
}

// An archive's event carries the one moment read under the lock, the one its write stamped into archived_at and
// updated_at: on a clock that moves at each read, the cycle's and the module's archive read it once and publish that
// reading (M7 P3c spec 2.6, R3).
func TestAnArchiveIsPublishedAtTheMomentItWrites(t *testing.T) {
	editor := shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor})
	w := sprintWorld(t)
	completedSprint(w)
	read := w.now
	cycles := &deletionEvents{lockWorld: w}
	if _, err := NewArchiveCycle(ArchiveCycleDeps{Locks: w.locks(), Cycles: w, Clock: ticking{w}, Events: cycles}).Execute(editor, lockCycle); err != nil {
		t.Fatal(err)
	}
	stored := w.updated[0]
	if !stored.ArchivedAt.Equal(read) || !stored.UpdatedAt.Equal(read) {
		t.Fatalf("stored %+v; want archived and changed at %v, the first reading", stored, read)
	}
	wantArchive(t, cycles.published, "planning.cycle_archived", shared.EntityRef{Kind: "cycle", ID: lockCycle}, *stored.ArchivedAt)
	if clockReads(w.calls) != 1 {
		t.Errorf("calls %q; want the clock read once", w.calls)
	}

	m := archiveWorld(domain.ModuleCompleted, false)
	read = m.now
	m.events = &deletionEvents{lockWorld: m.lockWorld}
	uc := NewArchiveModule(ArchiveModuleDeps{Locks: m.locks(), Modules: m, Clock: ticking{m.lockWorld}, Events: m.events})
	if _, err := uc.Execute(editor, lockModule); err != nil {
		t.Fatal(err)
	}
	if len(m.stored) != 1 || !m.stored[0].ArchivedAt.Equal(read) || !m.stored[0].UpdatedAt.Equal(read) {
		t.Fatalf("stored %+v; want archived and changed at %v, the first reading", m.stored, read)
	}
	wantArchive(t, m.events.published, "planning.module_archived", shared.EntityRef{Kind: "module", ID: lockModule}, *m.stored[0].ArchivedAt)
	if clockReads(m.calls) != 1 {
		t.Errorf("calls %q; want the clock read once", m.calls)
	}
}

// A publication that fails fails the archive: the transaction ends with its error, which rolls the write back; the
// cycle's and the module's alike (M7 P3c spec 2.6).
func TestAFailedPublicationFailsTheArchive(t *testing.T) {
	editor := shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor})
	refusal := errors.New("a subscriber failed")
	w := sprintWorld(t)
	completedSprint(w)
	cycles := &deletionEvents{lockWorld: w, refusal: refusal}
	if _, err := NewArchiveCycle(ArchiveCycleDeps{Locks: w.locks(), Cycles: w, Clock: w, Events: cycles}).Execute(editor, lockCycle); !errors.Is(err, refusal) {
		t.Fatalf("Execute() = %v, want the publication's error", err)
	}
	if want := []string{"SetCycleArchived true", "Publish planning.cycle_archived", "End", "Leave"}; !slices.Equal(w.calls[len(w.calls)-4:], want) {
		t.Errorf("calls %q, want them to end %q", w.calls, want)
	}

	m := archiveWorld(domain.ModuleCancelled, false)
	m.events = &deletionEvents{lockWorld: m.lockWorld, refusal: refusal}
	uc := NewArchiveModule(ArchiveModuleDeps{Locks: m.locks(), Modules: m, Clock: m, Events: m.events})
	if _, err := uc.Execute(editor, lockModule); !errors.Is(err, refusal) {
		t.Fatalf("Execute() = %v, want the publication's error", err)
	}
	if want := []string{"SetModuleArchived true", "Publish planning.module_archived", "End", "Leave"}; !slices.Equal(m.calls[len(m.calls)-4:], want) {
		t.Errorf("calls %q, want them to end %q", m.calls, want)
	}
}
