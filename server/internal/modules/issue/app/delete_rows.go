package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteRowsResult counts only committed writes; Complete excludes pending candidates.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
}

// RowStep names an issue-owned table in the deletion scan.
type RowStep string

// ProjectRemainder identifies the first unfinished table and its committed cursor.
type ProjectRemainder struct {
	ProjectID, After uuid.UUID
	Step             RowStep
}

// RemainingRows probes candidates without locking or writing at a budget boundary.
type RemainingRows interface {
	ProjectDeletionRemaining(context.Context, ProjectRemainder) (bool, error)
	WorkspaceDraftDeletionRemaining(context.Context, WorkspaceDraftBatch) (bool, error)
}

// DeleteRowsDeps supplies short job transactions and a monotonic runtime clock.
type DeleteRowsDeps struct {
	Rows       ProjectRowsDeleter
	Relations  *RelationDeletions
	Remaining  RemainingRows
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows drains deleted parents without taking a project lock or write gate.
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// Execute commits each batch before counting it or advancing the shared scan.
// RuntimeNow must retain time.Now's monotonic reading; at is audit data only.
func (u *DeleteRows) Execute(ctx context.Context, scope string, id uuid.UUID, at time.Time, by uuid.UUID) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	if (scope != "project" && scope != "workspace_drafts") || id == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() {
		return result, errors.New("invalid row deletion scope or audit data")
	}
	start := u.d.RuntimeNow()
	budgetReached := errors.New("row deletion budget reached")
	run := func(name RowStep, step func(context.Context, RowBatch) (BatchProgress, error)) error {
		return deleteRowBatches(ctx, func(ctx context.Context, b RowBatch) (BatchProgress, error) {
			if err := ctx.Err(); err != nil {
				return BatchProgress{}, err
			}
			if u.d.RuntimeNow().Sub(start) >= u.d.Budget {
				var pending bool
				var err error
				if scope == "project" {
					pending, err = u.d.Remaining.ProjectDeletionRemaining(ctx, ProjectRemainder{ProjectID: id, Step: name, After: b.After})
				} else {
					pending, err = u.d.Remaining.WorkspaceDraftDeletionRemaining(ctx, workspaceBatch(b))
				}
				if err != nil {
					return BatchProgress{}, err
				}
				result.Complete = !pending
				return BatchProgress{}, budgetReached
			}
			var progress BatchProgress
			err := u.d.Tx.WithinTx(ctx, func(tx context.Context) error {
				var err error
				progress, err = step(tx, b)
				return err
			})
			if err != nil {
				return BatchProgress{}, err
			}
			result.Deleted += progress.Written
			return progress, nil
		}, RowBatch{ProjectID: id, Limit: u.d.Batch, At: at, By: by})
	}
	var err error
	if scope == "workspace_drafts" {
		err = run("draft_issues", rowProgress(func(ctx context.Context, b RowBatch) ([]uuid.UUID, error) {
			return u.d.Rows.DeleteWorkspaceDraftRows(ctx, workspaceBatch(b))
		}))
	} else {
		names := DeletionRowSteps()
		for i, step := range rowSteps(u.d.Rows, u.d.Relations.ForProject) {
			if err = run(names[i], step); err != nil {
				break
			}
		}
	}
	if err == budgetReached {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	result.Complete = true
	return result, nil
}

func workspaceBatch(b RowBatch) WorkspaceDraftBatch {
	return WorkspaceDraftBatch{WorkspaceID: b.ProjectID, After: b.After, Limit: b.Limit, At: b.At, By: b.By}
}

// DeletionRowSteps is the shared scan and remainder order; callers own the slice.
func DeletionRowSteps() []RowStep {
	return []RowStep{"draft_issues", "issues", "issue_assignees", "issue_labels", "issue_relations", "issue_links", "issue_subscribers", "issue_mentions", "issue_comments", "issue_reactions", "comment_reactions", "issue_activities", "issue_description_versions"}
}
