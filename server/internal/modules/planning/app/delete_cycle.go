package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleDeleter deletes a cycle under its project's lock, in the transaction
// ctx carries: the cycle; its page's undeleted display settings, each
// account's row, in one statement locking them in id order before writing
// them (convention 5; M6 P3b spec 2.5). Each row deleted and last changed by
// by at at, and one deleted before keeps its deletion.
type CycleDeleter interface {
	DeleteCycle(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error
	DeleteCyclePreferenceRows(ctx context.Context, cycle uuid.UUID, at time.Time, by uuid.UUID) error
}

// DeleteCycleDeps are DeleteCycle's ports: Events publishes the deletion in
// its transaction, which the issue module's subscriber writes in too (M6
// P3a spec R6).
type DeleteCycleDeps struct {
	Locks  Locks
	Cycles CycleDeleter
	Clock  Clock
	Events shared.EventPublisher
}

// DeleteCycle answers deleteCycle (M6 P1 spec 2.3, 2.4).
type DeleteCycle struct{ d DeleteCycleDeps }

// NewDeleteCycle returns the use case over d.
func NewDeleteCycle(d DeleteCycleDeps) *DeleteCycle { return &DeleteCycle{d: d} }

// Execute takes the lock path on the cycle id deciding cycle.delete, which
// its project's admins pass and its creator too (AllowCreator); then under
// the lock, the clock, and the cycle and its page's display settings deleted
// and last changed by the caller at that moment, in the write's transaction
// (M6 P3b spec 2.5), whatever its status and archive (R5); then its deletion
// published in that transaction (deletionEvent, M6 P3a spec R6), whose
// failure fails the deletion.
func (u *DeleteCycle) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, write{cycle: &id, action: domain.ActionCycleDelete}, func(ctx context.Context, h held) error {
		at, by := u.d.Clock.Now(), actor.UserID
		if err := u.d.Cycles.DeleteCycle(ctx, id, at, by); err != nil {
			return err
		}
		if err := u.d.Cycles.DeleteCyclePreferenceRows(ctx, id, at, by); err != nil {
			return err
		}
		c := h.cycle
		return u.d.Events.Publish(ctx, deletionEvent(domain.EventCycleDeleted, shared.EntityRef{Kind: "cycle", ID: c.ID}, c.WorkspaceID, c.ProjectID,
			c.Name, at, by))
	})
}
