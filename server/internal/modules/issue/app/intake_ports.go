package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// IntakeListRead is a page of a project's intake list as the store reads
// it (M7 P2b spec 2.3): the project, the caller and whether his grant on
// it sees only his own work items, the validated query, and the moment of
// the request, at which each row's status is its effective one (R3).
type IntakeListRead struct {
	ProjectID, ActorID uuid.UUID
	OwnOnly            bool
	Query              domain.IntakeQuery
	Now                time.Time
}

// IntakeRecord is a row of an intake list's page: the work item as a list
// row has it, without its description and its attachments' count, with
// its intake row as stored (Issue.Intake); its position in the page's
// order; and the number of rows the filters select, the cursor aside.
type IntakeRecord struct {
	Issue    domain.Issue
	Position domain.IntakeCursor
	Total    int64
}

// IntakeLister reads a project's intake list (M7 P2b spec 2.3), in the
// read transaction ctx carries.
type IntakeLister interface {
	// ListIntake reads the page in.Query asks for and the row after it,
	// when there is one: the project's undeleted work items with an
	// undeleted intake row, accepted and archived ones too, those of
	// another creator left out when in.OwnOnly (an unknown creator too),
	// each filter applied, the statuses effective at in.Now, the days'
	// ranges half-open; in the order, equal keys by the work item's id the
	// same way, after the cursor. Each row's sub-items are counted as the
	// project list counts them: undeleted and unarchived, the caller's
	// alone when in.OwnOnly.
	ListIntake(ctx context.Context, in IntakeListRead) ([]IntakeRecord, error)
}

// IntakeCount is a project's number of pending intake work items as the
// caller counts them (M7 P2b spec 2.6).
type IntakeCount struct {
	ProjectID uuid.UUID
	Pending   int64
}

// IntakeCounter counts the pending work items of projects' intakes (M7
// P2b spec 2.6), in the read transaction ctx carries.
type IntakeCounter interface {
	// CountPendingIntake counts, in each of the projects, each given once,
	// its undeleted intake rows of undeleted work items whose effective
	// status at now is pending (R3); of a project whose grant sees only the
	// actor's own work items, those he created alone, an unknown creator's
	// none. Every project, in id order, one with none too.
	CountPendingIntake(ctx context.Context, actor uuid.UUID, projects []VisibleProject, now time.Time) ([]IntakeCount, error)
}

// IntakeProjects reads which projects have their intake turned on (M7 P2b
// spec 2.6, DP-20): project.Provide's, converted (bootstrap/ports.go).
type IntakeProjects interface {
	// IntakeOn reads which of the projects ids are undeleted, unarchived
	// and have their intake turned on, in id order, in the read transaction
	// ctx carries. It reads no membership: the caller gives the projects he
	// sees (VisibleProjects.Visible).
	IntakeOn(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error)
}
