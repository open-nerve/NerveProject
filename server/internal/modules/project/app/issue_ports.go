package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueCascade is what the project's deletions ask of the work items (M4
// design 3.3): the issue module implements it (issue.Provide). Each method
// runs in the transaction ctx carries, which the caller began and holds
// the project's or the workspace's lock in, and returns a failure as
// itself, so the caller's whole write rolls back; one that deletes writes
// by and now into the rows it deletes.
type IssueCascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows:
	// deleteProject's last step.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of
	// the projects: deleteWorkspace's last step, on the projects it locked
	// (LockWorkspaceProjects).
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state: deleteState's check, under the project's lock.
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabel soft-deletes the rows that put the labels on work items:
	// deleteLabel's last step, on the label and the labels under it.
	DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// IssueProject is a project as project.Provide's lock for the issue
// module answers it (M4 design 6.5, ProjectGate.Lock): its workspace,
// whether it is archived, its default assignee, nil for none, and its
// identifier. bootstrap converts it into issue's value.
type IssueProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
	Identifier        string
}

// IssueState is a state of a project as project.Provide's catalog for the
// issue module reads it (M4 design 6.5, StateCatalog): its id, its name,
// its group, and whether it is the project's default. bootstrap converts
// it into issue's value.
type IssueState struct {
	ID      uuid.UUID
	Name    string
	Group   string
	Default bool
}

// IssueLabel is a label of a project as project.Provide's catalog for the
// issue module reads it (M4 design 6.5, LabelCatalog): its id and its
// name. bootstrap converts it into issue's value.
type IssueLabel struct {
	ID   uuid.UUID
	Name string
}

// IssueProjectIdentifier is a project as project.Provide reads it for the
// issue module's getIssueByIdentifier (M4 design 3.7): its id and its
// identifier. bootstrap converts it into issue's value.
type IssueProjectIdentifier struct {
	ID         uuid.UUID
	Identifier string
}

// IssueMemberVisibility is an active member's role and effective guest scope.
type IssueMemberVisibility struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}

// HistoricalIssueProject is an actor's historical project role and guest scope.
type HistoricalIssueProject struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}

// VisibleIssueProject is an actor's ordinary project role and guest scope.
type VisibleIssueProject struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}
