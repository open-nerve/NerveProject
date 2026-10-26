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
	ID                  uuid.UUID
	WorkspaceID         uuid.UUID
	ProjectID           uuid.UUID
	SequenceID          int
	Name                string
	DescriptionHTML     string
	DescriptionStripped *string
	StateID             uuid.UUID
	Priority            domain.Priority
	StartDate           *shared.Date
	TargetDate          *shared.Date
	ParentID            *uuid.UUID
	SortOrder           float64
	CompletedAt         *time.Time
	By                  uuid.UUID
	At                  time.Time
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
// 3.7, 3.9): those it changes and the others as they are, and the moment
// and the account that change it last.
type IssueFields struct {
	ID                  uuid.UUID
	Name                string
	DescriptionHTML     string
	DescriptionStripped *string
	StateID             uuid.UUID
	Priority            domain.Priority
	SortOrder           float64
	StartDate           *shared.Date
	TargetDate          *shared.Date
	ParentID            *uuid.UUID
	CompletedAt         *time.Time
	By                  uuid.UUID
	At                  time.Time
}

// ParentRows are the work items an addition of sub-items puts under a
// work item (M4 design 3.9): IDs, in id order, under ParentID, each last
// changed at the moment and by the account given.
type ParentRows struct {
	IDs      []uuid.UUID
	ParentID uuid.UUID
	By       uuid.UUID
	At       time.Time
}

// ChangedRows are the work items a write changes last without a field of
// theirs (M4 design 3.7): the ends of a relation it adds or removes, IDs,
// in id order, each last changed at the moment and by the account given.
type ChangedRows struct {
	IDs []uuid.UUID
	By  uuid.UUID
	At  time.Time
}

// RelationDeletion is a relation's deletion as it is written (M4 design
// 3.10): the relation ID, soft-deleted at the moment and by the account
// given.
type RelationDeletion struct {
	ID uuid.UUID
	By uuid.UUID
	At time.Time
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
	ActorID     *uuid.UUID
	At          time.Time
}
