package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Clock is the business clock.
type Clock interface {
	Now() time.Time
}

// CycleStore is the cycles' store as the writes use it, in the transaction
// ctx carries: the cycle the lock reads, the project's other cycles' dates
// and the lowest of their sort orders, all undeleted, archived ones too, and
// the creation of a cycle.
type CycleStore interface {
	CycleReader
	ProjectCycleDates(ctx context.Context, project uuid.UUID) ([]domain.DatedCycle, error)
	LowestCycleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error)
	CreateCycle(ctx context.Context, c domain.Cycle) error
}

// CreateCycleDeps are what CreateCycle needs.
type CreateCycleDeps struct {
	Locks  Locks
	Cycles CycleStore
	Clock  Clock
}

// CreateCycle creates a cycle in a project (createCycle).
type CreateCycle struct{ d CreateCycleDeps }

// NewCreateCycle returns the use case over d.
func NewCreateCycle(d CreateCycleDeps) *CreateCycle { return &CreateCycle{d: d} }

// Execute checks in's values, takes the project's lock and decision
// (Locks), refuses dates that share a day with another cycle of the project,
// and writes the cycle first among them, owned, created and last changed by
// the caller at the moment read after the lock.
func (u *CreateCycle) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewCycle) (domain.ReadCycle, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	if in, err = domain.CheckNewCycle(in); err != nil {
		return domain.ReadCycle{}, err
	}
	var out domain.ReadCycle
	err = u.d.Locks.run(ctx, actor, write{project: projectID, action: domain.ActionCycleWrite}, func(ctx context.Context, h held) error {
		if in.Dates.Dated() {
			others, err := u.d.Cycles.ProjectCycleDates(ctx, projectID)
			if err != nil {
				return err
			}
			if domain.Overlapping(uuid.UUID{}, in.Dates, others) {
				return domain.ErrDatesOverlap
			}
		}
		lowest, err := u.d.Cycles.LowestCycleSortOrder(ctx, projectID)
		if err != nil {
			return err
		}
		now := u.d.Clock.Now()
		by := actor.UserID
		c := domain.Cycle{
			ID: uuid.NewV7(), ProjectID: projectID, WorkspaceID: h.project.WorkspaceID, Name: in.Name, Description: in.Description,
			Dates: in.Dates, OwnedByID: &by, SortOrder: domain.NextSortOrder(lowest), CreatedAt: now, UpdatedAt: now,
			CreatedByID: &by, UpdatedByID: &by,
		}
		if err := u.d.Cycles.CreateCycle(ctx, c); err != nil {
			return err
		}
		out, err = readCycle(c, h.project.Timezone, now)
		return err
	})
	return out, err
}

// readCycle is c with its status on now's day in the project's time zone.
func readCycle(c domain.Cycle, timezone string, now time.Time) (domain.ReadCycle, error) {
	today, err := domain.Today(now, timezone)
	if err != nil {
		return domain.ReadCycle{}, err
	}
	return domain.ReadCycle{Cycle: c, Status: domain.StatusOf(c.Dates, today)}, nil
}
