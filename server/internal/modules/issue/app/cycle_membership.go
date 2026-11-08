package app

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// cycleChange is a write's change of the cycle a work item is in, as its
// activity names it (M6 P3a spec 2.4): the cycle it leaves and the one it
// joins, each nil for none.
type cycleChange = domain.Changed[*domain.Named]

// readCycleChange reads what planning holds about a write's change of the
// work item's cycle, from the cycle from to the cycle to, each nil for
// none, in the project projectID (M6 P3a spec R1–R3): once, under the
// project's lock the write holds, before its clock. The project's cycles
// turned off refuse any change, leaving one too (409
// planning.cycles_disabled, design 3.8); then the cycle joined, archived,
// refuses it (409 planning.archived). The cycle joined that is not the
// project's undeleted cycle is unknown: the write's one 422 names it at
// cycle_id (domain.CycleNotAllowed), with its other problems. The cycle
// left is not checked (R3), only named; it is the project's undeleted
// cycle, else an error: a cycle's deletion takes its work items out of it.
func readCycleChange(ctx context.Context, planning PlanningTargets, projectID uuid.UUID, from, to *uuid.UUID) (change cycleChange, unknown bool,
	err error) {
	var asked []uuid.UUID
	for _, id := range []*uuid.UUID{from, to} {
		if id != nil {
			asked = append(asked, *id)
		}
	}
	facts, err := planning.Targets(ctx, projectID, ascending(asked), nil)
	if err != nil {
		return cycleChange{}, false, err
	}
	if !facts.CyclesEnabled {
		return cycleChange{}, false, domain.ErrCyclesDisabled
	}
	if to != nil {
		joined, found := cycleOf(facts.Cycles, *to)
		switch {
		case !found:
			unknown = true
		case joined.Archived:
			return cycleChange{}, false, domain.ErrPlanningArchived
		}
		change.New = &domain.Named{ID: *to, Name: joined.Name}
	}
	if from != nil {
		left, found := cycleOf(facts.Cycles, *from)
		if !found {
			return cycleChange{}, false, fmt.Errorf("work item's cycle %s is not an undeleted cycle of project %s", *from, projectID)
		}
		change.Old = &domain.Named{ID: *from, Name: left.Name}
	}
	return change, unknown, nil
}

// cycleOf is the cycle id among cycles; found is false when it is not there.
func cycleOf(cycles []PlanningTarget, id uuid.UUID) (PlanningTarget, bool) {
	at := slices.IndexFunc(cycles, func(c PlanningTarget) bool { return c.ID == id })
	if at < 0 {
		return PlanningTarget{}, false
	}
	return cycles[at], true
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
