package app

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// cycleChange is a write's change of the cycle a work item is in, as its
// activity names it (M6 P3a spec 2.4): the cycle it leaves and the one it
// joins, each nil for none.
type cycleChange = domain.Changed[*domain.Named]

// namedCycleChange is c, a write's change of the work item's cycle in the
// project projectID, named by cycles, planning's answer about the cycles it
// leaves and joins (readMembership, M6 P3a spec R1–R3). The cycle joined,
// archived, refuses it (409 planning.archived); one that is not among
// cycles, the project's undeleted cycles, is unknown, for the write's 422
// at cycle_id. The cycle left is not checked (R3), only named; it is the
// project's undeleted cycle, else an error: a cycle's deletion takes its
// work items out of it.
func namedCycleChange(cycles []PlanningTarget, projectID uuid.UUID, c domain.CycleChange) (change cycleChange, unknown bool, err error) {
	if c.To != nil {
		joined, found := targetOf(cycles, *c.To)
		switch {
		case !found:
			unknown = true
		case joined.Archived:
			return cycleChange{}, false, domain.ErrPlanningArchived
		}
		change.New = &domain.Named{ID: *c.To, Name: joined.Name}
	}
	if c.From != nil {
		left, found := targetOf(cycles, *c.From)
		if !found {
			return cycleChange{}, false, fmt.Errorf("work item's cycle %s is not an undeleted cycle of project %s", *c.From, projectID)
		}
		change.Old = &domain.Named{ID: *c.From, Name: left.Name}
	}
	return change, unknown, nil
}

// refuseEnded refuses a write's joining the cycle joined, nil for none,
// that has ended on the day at falls on in the project's time zone (409
// planning.cycle_completed, M6 P3a spec R2): at is the write's clock, read
// after its last lock, so the day is the write's. A cycle the work item
// leaves is not checked (R3).
func refuseEnded(ctx context.Context, planning PlanningTargets, projectID uuid.UUID, joined *domain.Named, at time.Time) error {
	if joined == nil {
		return nil
	}
	ended, err := planning.CycleCompleted(ctx, projectID, joined.ID, at)
	switch {
	case err != nil:
		return err
	case ended:
		return domain.ErrCycleCompleted
	}
	return nil
}

// cycleRows are the rows a change of the work item's cycle writes (M6 P3a
// spec R4): the row of the cycle it leaves soft-deleted, then one of the
// cycle it joins inserted, each at the write's moment, by its writer; the
// row is not changed in place.
func cycleRows(c cycleChange) domain.Diff {
	var d domain.Diff
	if c.Old != nil {
		d.Removed = []uuid.UUID{c.Old.ID}
	}
	if c.New != nil {
		d.Added = []uuid.UUID{c.New.ID}
	}
	return d
}
