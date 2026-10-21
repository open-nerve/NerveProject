package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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
	StartDate   *shared.Date
	TargetDate  *shared.Date
	ParentID    *uuid.UUID
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

// IssueFields are a work item's fields as an update writes them (M4 design
// 3.7): those it changes and the others as they are, and the moment and
// the account that change it last.
type IssueFields struct {
	ID          uuid.UUID
	Name        string
	StateID     uuid.UUID
	Priority    domain.Priority
	StartDate   *shared.Date
	TargetDate  *shared.Date
	CompletedAt *time.Time
	By          uuid.UUID
	At          time.Time
}

// IssueArchival is a work item's archiving or unarchiving as it is written
// (M4 design 3.8): its archived_at, the moment, or nil for none, and the
// moment and the account that change it last.
type IssueArchival struct {
	ID         uuid.UUID
	ArchivedAt *time.Time
	By         uuid.UUID
	At         time.Time
}

// UnlinkRows are the rows of one of a work item's tables of accounts or
// labels, its assignees' or labels', that an update removes: its undeleted
// rows of Targets, soft-deleted at the moment and by the account given.
type UnlinkRows struct {
	IssueID uuid.UUID
	Targets []uuid.UUID
	By      uuid.UUID
	At      time.Time
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
