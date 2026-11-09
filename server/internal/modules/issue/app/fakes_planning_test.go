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
// each call logged: web's cycles turned off when off and its modules when
// modulesOff, its undeleted cycles cycles and modules modules, and
// completed the cycles that have ended on any day asked about.
type fakePlanning struct {
	log             *callLog
	off, modulesOff bool
	cycles, modules []app.PlanningTarget
	completed       []uuid.UUID
}

func (f *fakePlanning) Targets(ctx context.Context, projectID uuid.UUID, cycles, modules []uuid.UUID) (app.PlanningFacts, error) {
	if err := f.log.add(ctx, "Targets", projectID, cycles, modules); err != nil {
		return app.PlanningFacts{}, err
	}
	return app.PlanningFacts{CyclesEnabled: !f.off, ModulesEnabled: !f.modulesOff, Cycles: among(f.cycles, cycles), Modules: among(f.modules, modules)}, nil
}

// among are the targets of known that ids names, in known's order.
func among(known []app.PlanningTarget, ids []uuid.UUID) []app.PlanningTarget {
	return slices.DeleteFunc(slices.Clone(known), func(t app.PlanningTarget) bool { return !slices.Contains(ids, t.ID) })
}

// fakeDirectory is planning's directory of a list's cycles and modules
// (PlanningDirectory, M6 P3b spec R2): the names it holds of the projects
// the list's other directories are of, each read logged among the list's
// reads by ports, which fails the one it is told to.
type fakeDirectory struct {
	ports           *listingPorts
	cycles, modules []app.PlanningName
}

func (f fakeDirectory) CycleNames(ctx context.Context, projectIDs []uuid.UUID) ([]app.PlanningName, error) {
	return f.names(ctx, "cycles", projectIDs, f.cycles)
}

func (f fakeDirectory) ModuleNames(ctx context.Context, projectIDs []uuid.UUID) ([]app.PlanningName, error) {
	return f.names(ctx, "modules", projectIDs, f.modules)
}

func (f fakeDirectory) names(ctx context.Context, read string, projectIDs []uuid.UUID, names []app.PlanningName) ([]app.PlanningName, error) {
	f.ports.t.Helper()
	if !slices.Equal(projectIDs, f.ports.pids) {
		f.ports.t.Fatalf("%s scope %v; want %v", read, projectIDs, f.ports.pids)
	}
	return names, f.ports.called(ctx, read)
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

// InsertModules puts the work item in the modules r names, by its last
// writer, then (link); one it is in already is an error, as the partial
// unique key on module_issues (issue_id, module_id) makes it.
func (f *fakeIssues) InsertModules(ctx context.Context, r app.LinkRows) error {
	added, err := f.link(ctx, "InsertModules", r)
	switch {
	case err != nil:
		return err
	case slices.ContainsFunc(added, func(id uuid.UUID) bool { return slices.Contains(f.modules, id) }):
		return fmt.Errorf("InsertModules: %v into a work item in %v", added, f.modules)
	}
	f.modules = append(f.modules, added...)
	return nil
}

// RemoveModules takes the work item out of the modules r names, by its
// last writer, then (unlink).
func (f *fakeIssues) RemoveModules(ctx context.Context, r app.UnlinkRows) error {
	left, err := f.unlink(ctx, "RemoveModules", r, f.modules)
	if err == nil {
		f.modules = left
	}
	return err
}
