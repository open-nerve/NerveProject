package app

import (
	"context"
	"errors"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleBatch is one batch of a deleted project's cycles: at most Limit of its
// undeleted cycles whose ids are greater than After, in id order,
// soft-deleted at At by By.
type CycleBatch struct {
	ProjectID, After, By uuid.UUID
	Limit                int
	At                   time.Time
}

// CycleRows soft-deletes a deleted project's cycles a batch at a time, in the
// transaction ctx carries, locking the batch's rows in id order before it
// writes them; and, without locking or writing, tells whether undeleted
// cycles remain after a cursor.
type CycleRows interface {
	DeleteProjectCycles(ctx context.Context, b CycleBatch) ([]uuid.UUID, error)
	ProjectCyclesRemaining(ctx context.Context, project, after uuid.UUID) (bool, error)
}

// DeleteRowsResult counts only committed writes; Complete is false while
// undeleted cycles remain.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
}

// DeleteRowsDeps supplies short job transactions and a monotonic runtime clock.
type DeleteRowsDeps struct {
	Rows       CycleRows
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows soft-deletes a deleted project's cycles within a budget. It takes
// no project lock or write gate: the project is deleted, and no write reaches
// its cycles any more (M6 P1 spec 2.4).
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// Execute commits each batch in its own transaction before counting it or
// moving the cursor; only an empty batch ends the work. At the budget it
// stops before the next batch, Complete telling whether cycles remain.
// RuntimeNow must keep time.Now's monotonic reading; at is audit data only.
func (u *DeleteRows) Execute(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	if project == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() {
		return result, errors.New("invalid cycle deletion project or audit data")
	}
	start := u.d.RuntimeNow()
	b := CycleBatch{ProjectID: project, Limit: u.d.Batch, At: at, By: by}
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if u.d.RuntimeNow().Sub(start) >= u.d.Budget {
			pending, err := u.d.Rows.ProjectCyclesRemaining(ctx, project, b.After)
			if err != nil {
				return result, err
			}
			result.Complete = !pending
			return result, nil
		}
		var deleted []uuid.UUID
		err := u.d.Tx.WithinTx(ctx, func(tx context.Context) error {
			var err error
			deleted, err = u.d.Rows.DeleteProjectCycles(tx, b)
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
