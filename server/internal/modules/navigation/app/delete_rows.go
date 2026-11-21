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
// undeleted rows may remain.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
}

// DeleteRowsDeps supplies the tables' batches, short job transactions and a
// monotonic runtime clock.
type DeleteRowsDeps struct {
	Views      ViewDeletions
	Favorites  FavoriteDeletions
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows soft-deletes the views and the favorites of a deleted project,
// or of a deleted workspace's own, within a budget (P3b spec 2.7, P3c spec
// 2.6, R11). It waits for nothing, the work items' deletion least of all,
// and takes no lock of the project or the workspace: the parent is
// deleted, so no write reaches its rows any more (each authorizes on the
// parent first).
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// rowStep is one step of a deletion: a table's batch statement for the
// scope.
type rowStep func(context.Context, RowBatch) ([]uuid.UUID, error)

// steps are scope's steps in order, none for an unknown scope: the views,
// then the favorites; a workspace's own favorites its folders first, then
// its others, the total order of favorite rows (P3c spec 2.8).
func (u *DeleteRows) steps(scope DeletionScope) []rowStep {
	switch scope {
	case ProjectScope:
		return []rowStep{u.d.Views.DeleteProjectViews, u.d.Favorites.DeleteProjectFavorites}
	case WorkspaceScope:
		return []rowStep{u.d.Views.DeleteWorkspaceViews, u.d.Favorites.DeleteWorkspaceFavoriteFolders, u.d.Favorites.DeleteWorkspaceFavoriteItems}
	}
	return nil
}

// Execute runs the scope's steps in order, each with its own cursor from
// the start, under one budget. It commits each batch in its own
// transaction before counting it or moving its step's cursor; only an
// empty batch ends a step, and the last step's completes the work. At the
// budget it stops before the next batch, incomplete; a later run scans
// each step from the start again, a row deleted before being passed over.
// RuntimeNow must keep time.Now's monotonic reading; at and by are the
// deletion's, the rows' audit data.
func (u *DeleteRows) Execute(ctx context.Context, scope DeletionScope, id uuid.UUID, at time.Time, by uuid.UUID) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	steps := u.steps(scope)
	if steps == nil || id == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() {
		return result, errors.New("invalid navigation row deletion scope, parent or audit data")
	}
	start := u.d.RuntimeNow()
	for _, deleteBatch := range steps {
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
				break
			}
			result.Deleted += len(deleted)
			b.After = slices.MaxFunc(deleted, uuid.UUID.Compare)
		}
	}
	result.Complete = true
	return result, nil
}
