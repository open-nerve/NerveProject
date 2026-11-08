package app

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueTarget is a cycle or a module of a project as a work item's write
// names it (M6 P3a spec R1): its id, its name, which the write's activity
// names it by, and whether it is archived, which refuses joining it.
type IssueTarget struct {
	ID       uuid.UUID
	Name     string
	Archived bool
}

// IssueTargetFacts are what IssueTargets.Targets answers about a project:
// whether its cycles and its modules are turned on (M6 design 3.8), and,
// of the cycles and of the modules asked about, those that are its own and
// undeleted, archived or not, each in id order. One asked about and not
// among them is another project's, deleted, or none.
type IssueTargetFacts struct {
	CyclesEnabled, ModulesEnabled bool
	Cycles, Modules               []IssueTarget
}

// IssueTargetRows are the reads of IssueTargets, in the transaction ctx
// carries: the undeleted cycle (CycleReader); of the cycles or the modules
// ids, the project's undeleted ones, archived or not, in id order, another
// project's, a deleted one and an id of none not among them; and the ids,
// in id order, of the project's undeleted cycles, archived or not, that have
// not ended on today, without dates or ending on today or later, and of its
// undeleted modules, archived or not and in any status, whose target day
// has not passed on today, without one or on today or later.
type IssueTargetRows interface {
	CycleReader
	CycleTargets(ctx context.Context, project uuid.UUID, ids []uuid.UUID) ([]IssueTarget, error)
	ModuleTargets(ctx context.Context, project uuid.UUID, ids []uuid.UUID) ([]IssueTarget, error)
	OpenCycleIDs(ctx context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error)
	OpenModuleIDs(ctx context.Context, project uuid.UUID, today shared.Date) ([]uuid.UUID, error)
}

// IssueTargets is what the module tells the issue module about the cycles
// and modules of a project whose row lock the asking transaction holds: a
// work item's write, for the cycle and modules it names (M6 P3a spec R1),
// or the automatic archival of the project's work items, for the cycles
// and modules that keep theirs (R9). Every planning write takes that lock
// first, so what is read stays as read until the asking transaction
// commits; the project's facts too, read without a lock of their own. The
// toggles, a cycle's end and the day it is read on are the module's rules,
// here alone. planning.Provide hands it to bootstrap.
type IssueTargets struct {
	projects FactsReader
	rows     IssueTargetRows
}

// NewIssueTargets returns the issue targets over the projects' facts and the
// planning rows.
func NewIssueTargets(projects FactsReader, rows IssueTargetRows) *IssueTargets {
	return &IssueTargets{projects: projects, rows: rows}
}

// Targets reads the project's facts, then, when any is asked about, the
// project's undeleted cycles among cycles, then its undeleted modules among
// modules (IssueTargetFacts).
func (t *IssueTargets) Targets(ctx context.Context, project uuid.UUID, cycles, modules []uuid.UUID) (IssueTargetFacts, error) {
	f, err := t.facts(ctx, project)
	if err != nil {
		return IssueTargetFacts{}, err
	}
	out := IssueTargetFacts{CyclesEnabled: f.CycleView, ModulesEnabled: f.ModuleView}
	if len(cycles) > 0 {
		if out.Cycles, err = t.rows.CycleTargets(ctx, project, cycles); err != nil {
			return IssueTargetFacts{}, err
		}
	}
	if len(modules) > 0 {
		if out.Modules, err = t.rows.ModuleTargets(ctx, project, modules); err != nil {
			return IssueTargetFacts{}, err
		}
	}
	return out, nil
}

// CycleCompleted reports whether the cycle cycle of the project has ended on
// the day now falls on in the project's time zone (domain.Today): its
// status that day is completed (domain.StatusOf); one without dates, not
// begun or in progress has not. The asking write holds the project's lock
// and names a cycle Targets answered as the project's: a cycle that is not
// the project's undeleted one, a project that is not there and a time zone
// that does not load are internal errors.
func (t *IssueTargets) CycleCompleted(ctx context.Context, project, cycle uuid.UUID, now time.Time) (bool, error) {
	c, found, err := t.rows.GetCycle(ctx, cycle)
	switch {
	case err != nil:
		return false, err
	case !found || c.ProjectID != project:
		return false, fmt.Errorf("cycle %s is not an undeleted cycle of project %s", cycle, project)
	}
	today, err := t.today(ctx, project, now)
	if err != nil {
		return false, err
	}
	return domain.StatusOf(c.Dates, today) == domain.StatusCompleted, nil
}

// OpenTargets reads, on the day now falls on in the project's time zone,
// the ids of its undeleted cycles that have not ended and of its undeleted
// modules whose target day has not passed (IssueTargetRows), archived or
// not, a module in any status (M6 design 7, as Plane).
func (t *IssueTargets) OpenTargets(ctx context.Context, project uuid.UUID, now time.Time) (cycles, modules []uuid.UUID, err error) {
	today, err := t.today(ctx, project, now)
	if err != nil {
		return nil, nil, err
	}
	if cycles, err = t.rows.OpenCycleIDs(ctx, project, today); err != nil {
		return nil, nil, err
	}
	if modules, err = t.rows.OpenModuleIDs(ctx, project, today); err != nil {
		return nil, nil, err
	}
	return cycles, modules, nil
}

// today is the day now falls on in the project's time zone (domain.Today).
func (t *IssueTargets) today(ctx context.Context, project uuid.UUID, now time.Time) (shared.Date, error) {
	f, err := t.facts(ctx, project)
	if err != nil {
		return shared.Date{}, err
	}
	return domain.Today(now, f.Timezone)
}

// facts reads the project's facts. The asking transaction holds its lock,
// so it is there: one that is not is an internal error.
func (t *IssueTargets) facts(ctx context.Context, project uuid.UUID) (ProjectFacts, error) {
	f, found, err := t.projects.Facts(ctx, project)
	switch {
	case err != nil:
		return ProjectFacts{}, err
	case !found:
		return ProjectFacts{}, fmt.Errorf("project %s, whose lock the issue targets' caller holds, is not there", project)
	}
	return f, nil
}
