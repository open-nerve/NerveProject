package app

import (
	"context"
	"time"
	"uuid"
)

// Cascade carries out the project module's deletions on the work items
// (project's IssueCascade port, M4 design 3.3): it soft-deletes a deleted
// project's work-item-side rows in the deletion's transaction, at its
// moment and by its account, through the statements' orchestration
// (deleteProjectRows).
type Cascade struct {
	rows  ProjectRowsDeleter
	batch int
}

// NewCascade returns the cascade over rows, batch rows a statement.
func NewCascade(rows ProjectRowsDeleter, batch int) *Cascade {
	return &Cascade{rows: rows, batch: batch}
}

// DeleteProjectIssues soft-deletes the project's work-item-side rows:
// deleteProject's last step.
func (c *Cascade) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return deleteProjectRows(ctx, c.rows, c.batch, projectID, by, now)
}

// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of the
// projects, in their order: those deleteWorkspace locked and deletes.
func (c *Cascade) DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	for _, project := range projectIDs {
		if err := deleteProjectRows(ctx, c.rows, c.batch, project, by, now); err != nil {
			return err
		}
	}
	return nil
}
