package app

import (
	"context"
	"time"
	"uuid"
)

// PlanningTarget is a cycle or a module of the work item's project as
// planning answers it to a write that names it (M6 P3a spec R1): its id,
// its name, which the write's activity rows name it by (spec 2.4), and
// whether it is archived, which refuses joining it (R3).
type PlanningTarget struct {
	ID       uuid.UUID
	Name     string
	Archived bool
}

// PlanningFacts are what PlanningTargets.Targets answers about a project:
// whether its cycles and its modules are turned on (M6 design 3.8), and, of
// the cycles and of the modules asked about, those that are the project's
// and undeleted, archived or not, each in id order. One asked about and not
// among them is another project's, deleted, or none: a write may not join
// it (R2).
type PlanningFacts struct {
	CyclesEnabled, ModulesEnabled bool
	Cycles, Modules               []PlanningTarget
}

// PlanningTargets reads what planning holds about the cycle and modules a
// work item's write names (M6 P3a spec R1), in the write's transaction,
// under the project's row lock the write holds, which every write on the
// project's cycles and modules takes first, so what is read stays as read
// until the write commits. The toggles and a cycle's end are planning's
// rules: planning.Provide's IssueTargets, converted (bootstrap/ports.go).
type PlanningTargets interface {
	// Targets reads the project's toggles and, among cycles and modules,
	// its undeleted ones (PlanningFacts).
	Targets(ctx context.Context, projectID uuid.UUID, cycles, modules []uuid.UUID) (PlanningFacts, error)
	// CycleCompleted reports whether the cycle cycleID, which Targets
	// answered as the project's, has ended on the day now falls on in the
	// project's time zone; one without dates, not begun or in progress
	// has not. now is the write's clock, read after its last lock (R2).
	CycleCompleted(ctx context.Context, projectID, cycleID uuid.UUID, now time.Time) (bool, error)
}

// OpenPlanning reads the cycles and modules that keep a project's work
// items from automatic archival (M6 P3a spec R9, design 7), in the
// archival's transaction, under the project's row lock it holds, on the
// day now, its clock, falls on in the project's time zone: the project's
// undeleted cycles that have not ended, without dates or ending on that day
// or later, and its undeleted modules whose target day has not passed,
// without one or on that day or later; archived or not, a module in any
// status (as Plane). Each in id order. planning.Provide's IssueTargets,
// converted (bootstrap/ports.go).
type OpenPlanning interface {
	OpenTargets(ctx context.Context, projectID uuid.UUID, now time.Time) (cycles, modules []uuid.UUID, err error)
}

// PlanningName is a cycle or a module as PlanningDirectory answers it: its
// id and its name, which a list ordered by the cycles' or the modules'
// names keys its work items by (NameOrderKey, M6 P3b spec R2).
type PlanningName struct {
	ID   uuid.UUID
	Name string
}

// PlanningDirectory reads the names a list ordered by the cycles' or the
// modules' names needs (M6 P3b spec R2), in the read-only snapshot ctx
// carries, the one the list reads its rows in: of the projects projectIDs,
// every undeleted cycle, or module, archived or not, in id order. Another
// project's and a deleted one are not among them, nor is anything when no
// project is asked about. The projects' toggles do not matter, reading is
// not held to them (M6 design 3.8); the projects are those the list may
// read, whose cycles and modules any member, a guest too, may read.
// planning.Provide's IssueDirectory, converted (bootstrap/ports.go).
type PlanningDirectory interface {
	CycleNames(ctx context.Context, projectIDs []uuid.UUID) ([]PlanningName, error)
	ModuleNames(ctx context.Context, projectIDs []uuid.UUID) ([]PlanningName, error)
}
