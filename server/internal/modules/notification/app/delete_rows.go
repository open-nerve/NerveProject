package app

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// issueWaitLimit is how long after its enqueueing the deletion of a deleted
// project's notifications waits for the project's work items to be
// deleted; past it each run fails, and River's retries and attempt limit
// take over (M7 P1b spec 2.8): a River snooze consumes no attempt, so the
// wait alone would never end.
const issueWaitLimit = time.Hour

// DeleteRowsResult counts only committed writes. Waiting tells that the
// project's work items are still being deleted and nothing was written;
// Complete is false while undeleted notifications may remain.
type DeleteRowsResult struct {
	Deleted  int
	Complete bool
	Waiting  bool
}

// DeleteRowsDeps supplies the project's work items' read, short job
// transactions and a monotonic runtime clock.
type DeleteRowsDeps struct {
	Issues     ProjectIssues
	Rows       ProjectNotifications
	Tx         shared.TxManager
	RuntimeNow func() time.Time
	Batch      int
	Budget     time.Duration
}

// DeleteRows soft-deletes a deleted project's notifications within a
// budget, once none of the project's work items is undeleted (spec 2.8,
// R3): a fan-out writes a work item's notifications under the work item's
// row FOR SHARE and only while the work item is undeleted, and
// issue.delete_rows takes each row FOR NO KEY UPDATE as it deletes it, so
// once every one is deleted no fan-out writes a notification of the
// project any more. It takes no project lock.
type DeleteRows struct{ d DeleteRowsDeps }

// NewDeleteRows constructs the budgeted deletion task.
func NewDeleteRows(d DeleteRowsDeps) *DeleteRows { return &DeleteRows{d: d} }

// Execute first asks whether the project's work items remain: while they
// do, it writes nothing and answers Waiting, until issueWaitLimit after
// enqueued, the job's enqueueing, after which it fails. Then it commits each
// batch in its own transaction before counting it or moving the cursor;
// only an empty batch completes it. At the budget it stops before the next
// batch, incomplete. RuntimeNow must keep time.Now's monotonic reading; at
// and by are the deletion's, audit data only.
func (u *DeleteRows) Execute(ctx context.Context, project uuid.UUID, at time.Time, by uuid.UUID, enqueued time.Time) (DeleteRowsResult, error) {
	result := DeleteRowsResult{}
	if project == (uuid.UUID{}) || by == (uuid.UUID{}) || at.IsZero() || enqueued.IsZero() {
		return result, errors.New("invalid notification deletion project, audit data or enqueueing")
	}
	remain, err := u.d.Issues.ProjectIssuesRemain(ctx, project)
	if err != nil {
		return result, err
	}
	if remain {
		if waited := u.d.RuntimeNow().Sub(enqueued); waited >= issueWaitLimit {
			return result, fmt.Errorf("the work items of project %s are still not deleted %s after the deletion of its notifications was enqueued", project, waited)
		}
		result.Waiting = true
		return result, nil
	}
	start := u.d.RuntimeNow()
	b := RowBatch{ProjectID: project, Limit: u.d.Batch, At: at, By: by}
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
			deleted, err = u.d.Rows.DeleteProjectNotifications(tx, b)
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
