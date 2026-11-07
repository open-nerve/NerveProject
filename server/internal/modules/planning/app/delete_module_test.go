package app

import (
	"context"
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

// Deleting takes the lock path on the module, deciding module.delete, the module's creator in the target
// (AllowCreator); under the lock, the clock, then the module, its members' rows and its links' rows soft-deleted, each
// at that moment by the caller, in the write's transaction (M6 P2 spec 2.4, R15). Neither its status nor its archive
// holds it back.
func TestDeleteModule(t *testing.T) {
	for _, archived := range []bool{false, true} {
		w := moduleDeletions{archiveWorld(domain.ModuleInProgress, archived).lockWorld}
		uc := NewDeleteModule(DeleteModuleDeps{Locks: w.locks(), Modules: w, Clock: w})
		if err := uc.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: lockEditor}), lockModule); err != nil {
			t.Fatalf("archived=%t: %v", archived, err)
		}
		want := []string{"GetModule outside tx", "Facts outside tx", "Authorize module.delete outside tx", "Enter 1 outside tx", "Begin", "Lock",
			"GetModule", "Authorize module.delete", "Now", "DeleteModule true true true", "DeleteModuleMemberRows true true true",
			"DeleteModuleLinkRows true true true", "End", "Leave"}
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
