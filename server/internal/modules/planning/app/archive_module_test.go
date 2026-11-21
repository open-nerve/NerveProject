package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleArchives is the lock world with its module store for an archive: what SetModuleArchived stored; and its
// publisher, which the last execute was given.
type moduleArchives struct {
	*lockWorld
	stored []domain.Module
	events *deletionEvents
}

func (w *moduleArchives) SetModuleArchived(ctx context.Context, m domain.Module) error {
	w.log(ctx, fmt.Sprintf("SetModuleArchived %t", m.ArchivedAt != nil))
	w.stored = append(w.stored, m)
	return nil
}

// archiveWorld is the module world whose module is Auth, of status, made by its creator an hour before now, the 7th
// of November 2026 at noon; archived an hour before too when archived.
func archiveWorld(status domain.ModuleStatus, archived bool) *moduleArchives {
	w := newModuleWorld()
	made := time.Date(2026, 11, 7, 11, 0, 0, 0, time.UTC)
	m := w.modules[lockModule]
	m.Name, m.Status, m.CreatedAt, m.UpdatedAt, m.UpdatedByID = "Auth", status, made, made, &lockModuleCreator
	if archived {
		m.ArchivedAt = &made
	}
	w.modules[lockModule] = m
	w.now = time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	return &moduleArchives{lockWorld: w}
}

// execute archives the world's module, or unarchives it, as the editor.
func (w *moduleArchives) execute(archive bool) (domain.Module, error) {
	w.events = &deletionEvents{lockWorld: w.lockWorld}
	d := ArchiveModuleDeps{Locks: w.locks(), Modules: w, Clock: w, Events: w.events}
	uc := NewUnarchiveModule(d)
	if archive {
		uc = NewArchiveModule(d)
	}
	return uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule)
}

// Archiving a completed or a cancelled module takes the lock path on the module (Locks, module.write); under the lock,
// the clock, then the write: archived at that moment, last changed by the caller at the same moment, the rest as the
// lock read it; then the archive published in the write's transaction (P3c spec 2.6, R3). The answer is the module as
// stored.
func TestArchiveModule(t *testing.T) {
	for _, status := range []domain.ModuleStatus{domain.ModuleCompleted, domain.ModuleCancelled} {
		w := archiveWorld(status, false)
		got, err := w.execute(true)
		if err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		want := []string{"GetModule outside tx", "Facts outside tx", "Authorize module.write outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetModule", "Authorize module.write", "Now", "SetModuleArchived true", "Publish planning.module_archived", "End", "Leave"}
		if !slices.Equal(w.calls, want) {
			t.Fatalf("%s: calls\n%q\nwant\n%q", status, w.calls, want)
		}
		stored := w.modules[lockModule]
		stored.ArchivedAt, stored.UpdatedAt, stored.UpdatedByID = &w.now, w.now, &lockEditor
		if len(w.stored) != 1 || !reflect.DeepEqual(w.stored[0], stored) || !reflect.DeepEqual(got, stored) {
			t.Fatalf("%s: stored %+v, answered %+v\nwant %+v", status, w.stored, got, stored)
		}
		wantArchive(t, w.events.published, "planning.module_archived", shared.EntityRef{Kind: "module", ID: lockModule}, *w.stored[0].ArchivedAt)
	}
}

// Only a completed or a cancelled module is archived (planning.module_not_closed), refused after the decision and
// before the clock. An archived module archived again, and one not archived unarchived, whatever their status, answer
// as the lock read them and read no clock nor write nor publish anything (M6 P2 spec R3; P3c spec 2.6).
func TestArchiveModuleRefusesAndRepeats(t *testing.T) {
	for _, status := range []domain.ModuleStatus{domain.ModuleBacklog, domain.ModulePlanned, domain.ModuleInProgress, domain.ModulePaused} {
		w := archiveWorld(status, false)
		if _, err := w.execute(true); !errors.Is(err, domain.ErrModuleNotClosed) || len(w.stored) != 0 || slices.Contains(w.calls, "Now") ||
			!slices.Contains(w.calls, "Authorize module.write") || len(w.events.published) != 0 {
			t.Errorf("%s archived: %v, stored %v, calls %q; want planning.module_not_closed after the decision", status, err, w.stored, w.calls)
		}
	}
	for _, tt := range []struct {
		status  domain.ModuleStatus
		archive bool
	}{{domain.ModuleCompleted, true}, {domain.ModulePlanned, false}, {domain.ModuleCancelled, false}} {
		w := archiveWorld(tt.status, tt.archive)
		got, err := w.execute(tt.archive)
		if err != nil || len(w.stored) != 0 || slices.Contains(w.calls, "Now") || !reflect.DeepEqual(got, w.modules[lockModule]) ||
			len(w.events.published) != 0 {
			t.Errorf("%s archive=%t again: %+v, %v, stored %v, calls %q, published %v", tt.status, tt.archive, got, err, w.stored, w.calls,
				w.events.published)
		}
	}
}

// Unarchiving clears the archive, whatever the module's status, last changed by the caller at the moment read after
// the lock, and publishes nothing (P3c spec 2.6).
func TestUnarchiveModule(t *testing.T) {
	for _, status := range []domain.ModuleStatus{domain.ModuleCompleted, domain.ModuleCancelled, domain.ModulePlanned} {
		w := archiveWorld(status, true)
		got, err := w.execute(false)
		if err != nil {
			t.Fatalf("%s: %v", status, err)
		}
		stored := w.modules[lockModule]
		stored.ArchivedAt, stored.UpdatedAt, stored.UpdatedByID = nil, w.now, &lockEditor
		if len(w.stored) != 1 || !reflect.DeepEqual(w.stored[0], stored) || !reflect.DeepEqual(got, stored) ||
			!slices.Contains(w.calls, "SetModuleArchived false") || len(w.events.published) != 0 {
			t.Fatalf("%s: stored %+v, answered %+v, calls %q, published %v\nwant %+v", status, w.stored, got, w.calls, w.events.published, stored)
		}
	}
}
