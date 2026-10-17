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
