package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleArchiver writes a cycle's archive under its project's lock: its
// archived_at, and its last change.
type CycleArchiver interface {
	SetCycleArchived(ctx context.Context, c domain.Cycle) error
}

// ArchiveCycleDeps are ArchiveCycle's ports.
type ArchiveCycleDeps struct {
	Locks  Locks
	Cycles CycleArchiver
	Clock  Clock
}

// ArchiveCycle answers archiveCycle, or unarchiveCycle (M6 P1 spec 2.3, R3).
type ArchiveCycle struct {
	d       ArchiveCycleDeps
	archive bool
}

// NewArchiveCycle returns archiveCycle's use case over d.
func NewArchiveCycle(d ArchiveCycleDeps) *ArchiveCycle { return &ArchiveCycle{d: d, archive: true} }

// NewUnarchiveCycle returns unarchiveCycle's use case over d.
func NewUnarchiveCycle(d ArchiveCycleDeps) *ArchiveCycle { return &ArchiveCycle{d: d} }

// Execute takes the lock path on the cycle id (Locks); then under the lock:
// the clock; a cycle already as asked answers as read and writes nothing
// (R3); one to archive that is not completed on that moment's day in its
// project's time zone is refused (planning.cycle_not_completed); then its
// archived_at is that moment, or none, and its last change the caller's at
// that moment. The answer is the cycle as stored, with its status.
func (u *ArchiveCycle) Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	var out domain.ReadCycle
	err = u.d.Locks.run(ctx, actor, write{named: named{cycle: &id}, action: domain.ActionCycleWrite}, func(ctx context.Context, h held) error {
		now := u.d.Clock.Now()
		today, err := domain.Today(now, h.project.Timezone)
		if err != nil {
			return err
		}
		status := domain.StatusOf(h.cycle.Dates, today)
		c := h.cycle
		switch {
		case (c.ArchivedAt != nil) == u.archive:
			out = domain.ReadCycle{Cycle: c, Status: status}
			return nil
		case u.archive && status != domain.StatusCompleted:
			return domain.ErrCycleNotCompleted
		case u.archive:
			c.ArchivedAt = &now
		default:
			c.ArchivedAt = nil
		}
		by := actor.UserID
		c.UpdatedAt, c.UpdatedByID = now, &by
		if err := u.d.Cycles.SetCycleArchived(ctx, c); err != nil {
			return err
		}
		out = domain.ReadCycle{Cycle: c, Status: status}
		return nil
	})
	return out, err
}
