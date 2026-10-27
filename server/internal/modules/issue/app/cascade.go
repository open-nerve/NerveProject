package app

import (
	"context"
	"slices"
	"time"
	"uuid"
)

// StateLabelUses are the work items' uses of a project's states and
// labels, as the project module's deletions of them ask about them (M4
// design 3.6), each in the transaction ctx carries.
type StateLabelUses interface {
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state.
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabels soft-deletes the undeleted rows that put the labels on
	// work items, at now by by.
	DetachLabels(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Cascade carries out the project module's deletions on the work items
// (project's IssueCascade port, M4 design 3.3): it soft-deletes a deleted
// project's work-item-side rows in the deletion's transaction, at its
// moment and by its account, through the statements' orchestration
// (deleteProjectRows); it tells a state's deletion whether work items are
// in the state, and takes a deleted label off the work items.
type Cascade struct {
	rows  ProjectRowsDeleter
	uses  StateLabelUses
	batch int
}

// NewCascade returns the cascade over rows and uses, batch rows a
// statement.
func NewCascade(rows ProjectRowsDeleter, uses StateLabelUses, batch int) *Cascade {
	return &Cascade{rows: rows, uses: uses, batch: batch}
}

// DeleteProjectIssues soft-deletes the project's work-item-side rows:
// deleteProject's last step.
func (c *Cascade) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return deleteProjectRows(ctx, c.rows, c.batch, projectID, by, now)
}

// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each of the
// projects, in their order: those deleteWorkspace locked and deletes. Drafts
// without a project are deleted first, including in a workspace with no projects.
func (c *Cascade) DeleteWorkspaceIssues(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	b := WorkspaceDraftBatch{WorkspaceID: workspaceID, Limit: c.batch, By: by, At: now}
	for {
		ids, err := c.rows.DeleteWorkspaceDraftRows(ctx, b)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			break
		}
		b.After = slices.MaxFunc(ids, uuid.UUID.Compare)
	}
	for _, project := range projectIDs {
		if err := deleteProjectRows(ctx, c.rows, c.batch, project, by, now); err != nil {
			return err
		}
	}
	return nil
}

// StateInUse reports whether an undeleted work item, archived or not, is
// in the state: deleteState's check.
func (c *Cascade) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	return c.uses.StateInUse(ctx, stateID)
}

// DetachLabel soft-deletes the rows that put the labels on work items, at
// now by by: deleteLabel's last step.
func (c *Cascade) DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return c.uses.DetachLabels(ctx, labelIDs, by, now)
}
