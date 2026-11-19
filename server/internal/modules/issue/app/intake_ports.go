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
