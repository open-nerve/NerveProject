package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// moduleDeletions is the lock world with its module store for a deletion: each statement it ran, as "name module at
// by", the moment and the account each the world's or not.
type moduleDeletions struct{ *lockWorld }

func (w moduleDeletions) deletes(ctx context.Context, name string, id uuid.UUID, at time.Time, by uuid.UUID) error {
	w.log(ctx, fmt.Sprintf("%s %t %t %t", name, id == lockModule, at.Equal(w.now), by == lockEditor))
	return nil
}

func (w moduleDeletions) DeleteModule(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	return w.deletes(ctx, "DeleteModule", id, at, by)
}

func (w moduleDeletions) DeleteModuleMemberRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	return w.deletes(ctx, "DeleteModuleMemberRows", module, at, by)
}

func (w moduleDeletions) DeleteModuleLinkRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	return w.deletes(ctx, "DeleteModuleLinkRows", module, at, by)
}

func (w moduleDeletions) DeleteModulePreferenceRows(ctx context.Context, module uuid.UUID, at time.Time, by uuid.UUID) error {
	return w.deletes(ctx, "DeleteModulePreferenceRows", module, at, by)
}

// Deleting takes the lock path on the module, deciding module.delete, the module's creator in the target
// (AllowCreator); under the lock, the clock, then the module, its members' rows, its links' rows and its page's
// display settings' rows soft-deleted, each at that moment by the caller, in the write's transaction (M6 P2 spec 2.4,
// R15; P3b spec 2.5). Neither its status nor its archive holds it back.
func TestDeleteModule(t *testing.T) {
	for _, archived := range []bool{false, true} {
		w := moduleDeletions{archiveWorld(domain.ModuleInProgress, archived).lockWorld}
		uc := NewDeleteModule(DeleteModuleDeps{Locks: w.locks(), Modules: w, Clock: w, Events: &deletionEvents{lockWorld: w.lockWorld}})
		if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule); err != nil {
			t.Fatalf("archived=%t: %v", archived, err)
		}
		want := []string{"GetModule outside tx", "Facts outside tx", "Authorize module.delete outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetModule", "Authorize module.delete", "Now", "DeleteModule true true true", "DeleteModuleMemberRows true true true",
			"DeleteModuleLinkRows true true true", "DeleteModulePreferenceRows true true true", "Publish planning.module_deleted", "End", "Leave"}
		if !slices.Equal(w.calls, want) {
			t.Fatalf("archived=%t: calls\n%q\nwant\n%q", archived, w.calls, want)
		}
		for _, target := range w.targets {
			if target.CreatorID != lockModuleCreator {
				t.Fatalf("archived=%t: target %+v, want the module's creator", archived, target)
			}
		}
	}
}

// Deleting a module publishes its deletion in the write's transaction, after its rows', its members', its links' and
// its page's display settings' (M6 P3a spec 2.5, R6): the module, its workspace and project, the caller at the moment
// of the deletion, and its name before it as the one change. A publication that fails fails the deletion.
func TestDeleteModulePublishesItsDeletion(t *testing.T) {
	w := moduleDeletions{archiveWorld(domain.ModuleInProgress, false).lockWorld}
	events := &deletionEvents{lockWorld: w.lockWorld}
	uc := NewDeleteModule(DeleteModuleDeps{Locks: w.locks(), Modules: w, Clock: w, Events: events})
	if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule); err != nil {
		t.Fatal(err)
	}
	wantDeletion(t, events, "planning.module_deleted", shared.EntityRef{Kind: "module", ID: lockModule}, "Auth")

	w = moduleDeletions{archiveWorld(domain.ModuleInProgress, false).lockWorld}
	events = &deletionEvents{lockWorld: w.lockWorld, refusal: errors.New("a subscriber failed")}
	uc = NewDeleteModule(DeleteModuleDeps{Locks: w.locks(), Modules: w, Clock: w, Events: events})
	if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule); !errors.Is(err, events.refusal) {
		t.Fatalf("Execute() = %v, want the publication's error", err)
	}
	if want := []string{"DeleteModulePreferenceRows true true true", "Publish planning.module_deleted", "End", "Leave"}; !slices.Equal(w.calls[len(w.calls)-4:], want) {
		t.Errorf("calls %q, want them to end %q", w.calls, want)
	}
}
