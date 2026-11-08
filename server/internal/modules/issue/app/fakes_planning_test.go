package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// fakePlanning is planning's side of a work item's write (M6 P3a spec R1),
// each call logged: web's cycles turned off when off, its undeleted cycles
// cycles, and completed those that have ended on any day asked about.
type fakePlanning struct {
	log       *callLog
	off       bool
	cycles    []app.PlanningTarget
	completed []uuid.UUID
}

func (f *fakePlanning) Targets(ctx context.Context, projectID uuid.UUID, cycles, modules []uuid.UUID) (app.PlanningFacts, error) {
	if err := f.log.add(ctx, "Targets", projectID, cycles, modules); err != nil {
		return app.PlanningFacts{}, err
	}
	known := slices.DeleteFunc(slices.Clone(f.cycles), func(c app.PlanningTarget) bool { return !slices.Contains(cycles, c.ID) })
	return app.PlanningFacts{CyclesEnabled: !f.off, ModulesEnabled: true, Cycles: known}, nil
}

func (f *fakePlanning) CycleCompleted(ctx context.Context, projectID, cycleID uuid.UUID, now time.Time) (bool, error) {
	if err := f.log.add(ctx, "CycleCompleted", projectID, cycleID, now.Format(time.RFC3339Nano)); err != nil {
		return false, err
	}
	return slices.Contains(f.completed, cycleID), nil
}

// InsertCycle puts the work item in the one cycle r names, by its last
// writer, then (link); one already in a cycle is an error, as the partial
// unique key on cycle_issues.issue_id makes it.
func (f *fakeIssues) InsertCycle(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertCycle", r)
	switch {
	case err != nil:
		return err
	case len(added) != 1 || f.cycle != nil:
		return fmt.Errorf("InsertCycle: %v into a work item in %v", added, f.cycle)
	}
	f.cycle = &added[0]
	return nil
}

// RemoveCycle takes the work item out of its cycle, the one r names, by
// its last writer, then (unlink).
func (f *fakeIssues) RemoveCycle(ctx context.Context, r app.UnlinkRows) error {
	var in []uuid.UUID
	if f.cycle != nil {
		in = []uuid.UUID{*f.cycle}
	}
	left, err := f.unlink(ctx, "RemoveCycle", r, in)
	if err == nil && len(left) == 0 {
		f.cycle = nil
	}
	return err
}
