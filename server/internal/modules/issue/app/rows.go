package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// IssueRow is a work item a write inserts (M4 design 3.5): its values,
// and the moment and the account that create it.
type IssueRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	SequenceID  int
	Name        string
	StateID     uuid.UUID
	Priority    domain.Priority
	SortOrder   float64
	CompletedAt *time.Time
	By          uuid.UUID
	At          time.Time
}

// LinkRows are new rows of one of a work item's tables of accounts or
// labels, its assignees', labels' or subscribers': IDs[i] is the row of
// Targets[i], the account or the label, under the work item, at the moment
// and by the account given.
type LinkRows struct {
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	IDs         []uuid.UUID
	Targets     []uuid.UUID
	By          uuid.UUID
	At          time.Time
}

// ActivityRow is a new row of a work item's activity (M4 design 3.13):
// its id, the work item's, the row's values, and its actor, who writes it,
// at the moment given.
type ActivityRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	Activity    domain.Activity
	ActorID     uuid.UUID
	At          time.Time
}
