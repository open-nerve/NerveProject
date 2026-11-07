package app

import (
	"context"
	"errors"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RowBatch is one batch of a deleted project's rows of one planning table:
// at most Limit of its undeleted rows whose ids are greater than After, in id
// order, soft-deleted at At by By.
type RowBatch struct {
	ProjectID, After, By uuid.UUID
	Limit                int
	At                   time.Time
}

// RowStep names a planning table in the deletion's scan.
type RowStep string

// DeletionRowSteps is the deletion's scan and remainder order, each parent
// before its children (M6 P2 spec R16); callers own the slice.
func DeletionRowSteps() []RowStep {
	return []RowStep{"cycles", "modules", "module_members", "module_links"}
}

// ProjectRemainder is where a deletion stopped: the step it was in and the
// step's committed cursor.
type ProjectRemainder struct {
	ProjectID, After uuid.UUID
	Step             RowStep
}

// ProjectRows soft-deletes a deleted project's rows of each planning table a
// batch at a time, in the transaction ctx carries, locking the batch's rows
// FOR NO KEY UPDATE in id order before it writes them; and, without locking
// or writing, tells whether undeleted rows remain from a remainder on: in its
// step after its cursor, or in any later step.
type ProjectRows interface {
	DeleteProjectCycles(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectModules(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectModuleMembers(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectModuleLinks(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	ProjectDeletionRemaining(ctx context.Context, r ProjectRemainder) (bool, error)
}

// DeleteRowsResult counts only committed writes; Complete is false while
// undeleted rows remain.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
}

// DeleteRowsDeps supplies short job transactions and a monotonic runtime clock.
type DeleteRowsDeps struct {
	Rows       ProjectRows
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows soft-deletes a deleted project's planning rows within a budget,
// table by table in DeletionRowSteps' order. It takes no project lock or
// write gate: the project is deleted, and no write reaches its rows any more
// (M6 P1 spec 2.4, P2 spec 2.4).
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// Execute commits each batch in its own transaction before counting it or
// moving the cursor; only an empty batch ends a step. At the budget it stops
// before the next batch, Complete telling whether rows remain from the step
// it was in on. RuntimeNow must keep time.Now's monotonic reading; at is
// audit data only.
func (u *DeleteRows) Execute(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	if project == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() {
		return result, errors.New("invalid planning deletion project or audit data")
	}
	start := u.d.RuntimeNow()
	deletes := map[RowStep]func(context.Context, RowBatch) ([]uuid.UUID, error){
		"cycles": u.d.Rows.DeleteProjectCycles, "modules": u.d.Rows.DeleteProjectModules,
		"module_members": u.d.Rows.DeleteProjectModuleMembers, "module_links": u.d.Rows.DeleteProjectModuleLinks,
	}
	for _, step := range DeletionRowSteps() {
		b := RowBatch{ProjectID: project, Limit: u.d.Batch, At: at, By: by}
		for {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if u.d.RuntimeNow().Sub(start) >= u.d.Budget {
				pending, err := u.d.Rows.ProjectDeletionRemaining(ctx, ProjectRemainder{ProjectID: project, Step: step, After: b.After})
				if err != nil {
					return result, err
				}
				result.Complete = !pending
				return result, nil
			}
			var deleted []uuid.UUID
			err := u.d.Tx.WithinTx(ctx, func(tx context.Context) error {
				var err error
				deleted, err = deletes[step](tx, b)
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
