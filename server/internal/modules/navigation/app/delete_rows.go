package app

import (
	"context"
	"errors"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DeleteRowsResult counts only committed writes; Complete is false while
// undeleted views may remain.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
}

// DeleteRowsDeps supplies short job transactions and a monotonic runtime
// clock.
type DeleteRowsDeps struct {
	Rows       ViewDeletions
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows soft-deletes the views of a deleted project, or of a deleted
// workspace's own, within a budget (P3b spec 2.7). It waits for nothing,
// the work items' deletion least of all, and takes no lock of the project
// or the workspace: the parent is deleted, so no write reaches its views
// any more (each authorizes on the parent first).
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// Execute commits each batch in its own transaction before counting it or
// moving the cursor; only an empty batch completes it. At the budget it
// stops before the next batch, incomplete. RuntimeNow must keep time.Now's
// monotonic reading; at and by are the deletion's, the views' audit data.
func (u *DeleteRows) Execute(ctx context.Context, scope DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	deletes := map[DeletionScope]func(context.Context, RowBatch) ([]uuid.UUID, error){
		ProjectScope: u.d.Rows.DeleteProjectViews, WorkspaceScope: u.d.Rows.DeleteWorkspaceViews,
	}
	deleteBatch, known := deletes[scope]
	if !known || id == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() {
		return result, errors.New("invalid view deletion scope, parent or audit data")
	}
	start := u.d.RuntimeNow()
	b := RowBatch{ParentID: id, Limit: u.d.Batch, At: at, By: by}
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if u.d.RuntimeNow().Sub(start) >= u.d.Budget {
			return result, nil
		}
		var deleted []uuid.UUID
		err := u.d.Tx.WithinTx(ctx, func(tx context.Context) error {
			var err error
			deleted, err = deleteBatch(tx, b)
			return err
		})
		if err != nil {
			return result, err
		}
		if len(deleted) == 0 {
			result.Complete = true
			return result, nil
		}
		result.Deleted += len(deleted)
		b.After = slices.MaxFunc(deleted, uuid.UUID.Compare)
	}
}
