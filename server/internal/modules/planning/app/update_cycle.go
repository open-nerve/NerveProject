package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleUpdater is what a change of a cycle reads and writes under its
// project's lock: the other cycles' dates, and the cycle as changed.
type CycleUpdater interface {
	ProjectCycleDates(ctx context.Context, project uuid.UUID) ([]domain.DatedCycle, error)
	UpdateCycle(ctx context.Context, c domain.Cycle) error
}

// UpdateCycleDeps are UpdateCycle's ports.
type UpdateCycleDeps struct {
	Locks  Locks
	Cycles CycleUpdater
	Clock  Clock
}

// UpdateCycle answers updateCycle (M6 P1 spec 2.3, R2, R4, R5).
type UpdateCycle struct{ d UpdateCycleDeps }

// NewUpdateCycle returns the use case over d.
func NewUpdateCycle(d UpdateCycleDeps) *UpdateCycle { return &UpdateCycle{d: d} }

// Execute checks p's values, before anything is read; takes the lock path
// on the cycle id (Locks); then under the lock, in this order: an archived
// cycle refuses any change, one of nothing too (planning.archived); the
// clock; a cycle completed on that moment's day in its project's time zone
// refuses any field but the sort order, even given as it is
// (planning.cycle_completed); the dates p gives merged with the cycle's
// (422); a change of nothing writes nothing and answers the cycle as read;
// dates that change are held to the other cycles' (planning.cycle_dates_overlap),
// the cycle itself left out; then the cycle is written, last changed by the
// caller at that moment. The answer is the cycle as stored, with its status.
func (u *UpdateCycle) Execute(ctx context.Context, id uuid.UUID, p domain.CyclePatch) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	if p, err = domain.CheckCyclePatch(p); err != nil {
		return domain.ReadCycle{}, err
	}
	var out domain.ReadCycle
	err = u.d.Locks.run(ctx, actor, write{named: named{cycle: &id}, action: domain.ActionCycleWrite}, func(ctx context.Context, h held) error {
		if h.cycle.ArchivedAt != nil {
			return domain.ErrArchived
		}
		now := u.d.Clock.Now()
		today, err := domain.Today(now, h.project.Timezone)
		if err != nil {
			return err
		}
		if domain.StatusOf(h.cycle.Dates, today) == domain.StatusCompleted && !p.OnlySortOrder() {
			return domain.ErrCycleCompleted
		}
		next, changed, err := p.Apply(h.cycle)
		switch {
		case err != nil:
			return err
		case !changed:
			out = domain.ReadCycle{Cycle: h.cycle, Status: domain.StatusOf(h.cycle.Dates, today)}
			return nil
		}
		if !next.Dates.Equal(h.cycle.Dates) && next.Dates.Dated() {
			others, err := u.d.Cycles.ProjectCycleDates(ctx, h.projectID)
			if err != nil {
				return err
			}
			if domain.Overlapping(id, next.Dates, others) {
				return domain.ErrDatesOverlap
			}
		}
		by := actor.UserID
		next.UpdatedAt, next.UpdatedByID = now, &by
		if err := u.d.Cycles.UpdateCycle(ctx, next); err != nil {
			return err
		}
		out = domain.ReadCycle{Cycle: next, Status: domain.StatusOf(next.Dates, today)}
		return nil
	})
	return out, err
}
