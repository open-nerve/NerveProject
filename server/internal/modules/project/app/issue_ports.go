package app

import (
	"context"
	"time"
	"uuid"
)

// IssueCascade is what the project's deletions ask of the work items (M4
// design 3.3): the issue module implements it (issue.Provide). Each method
// runs in the transaction ctx carries, which the caller began and holds
// the project's or the workspace's lock in, writes by and now into the
// rows it deletes, and returns a failure as itself, so the caller's whole
// write rolls back.
type IssueCascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows:
	// deleteProject's last step.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of
	// the projects: deleteWorkspace's last step, on the projects it locked
	// (LockWorkspaceProjects).
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// IssueProject is a project as project.Provide's lock for the issue
// module answers it (M4 design 6.5, ProjectGate.Lock): its workspace,
// whether it is archived, and its default assignee, nil for none.
// bootstrap converts it into issue's value.
type IssueProject struct {
	WorkspaceID       uuid.UUID
	Archived          bool
	DefaultAssigneeID *uuid.UUID
}

// IssueState is a state of a project as project.Provide's catalog for the
// issue module reads it (M4 design 6.5, StateCatalog): its id, its group,
// and whether it is the project's default. bootstrap converts it into
// issue's value.
type IssueState struct {
	ID      uuid.UUID
	Group   string
	Default bool
}
