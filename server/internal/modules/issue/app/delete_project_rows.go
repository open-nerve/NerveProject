// Package app holds the issue module's use cases and the ports they need
// (M4 design 6.2).
package app

import (
	"context"
	"slices"
	"time"
	"uuid"
)

// RowBatch is one batch of a deleted project's work-item-side rows: at
// most Limit of its undeleted rows whose ids are greater than After, in id
// order, soft-deleted at At by By (M4 design 3.6).
type RowBatch struct {
	ProjectID, After uuid.UUID
	Limit            int
	By               uuid.UUID
	At               time.Time
}

// ProjectRowsDeleter soft-deletes a deleted project's rows of one table a
// batch at a time, in the transaction ctx carries; each method locks the
// batch's rows FOR NO KEY UPDATE in id order before it writes them
// (convention 5) and returns the ids it deleted.
type ProjectRowsDeleter interface {
	DeleteProjectIssueRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectAssigneeRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectLabelRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	// DeleteProjectRelationRows' batch is of the relations whose either
	// end is a work item of the project.
	DeleteProjectRelationRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectSubscriberRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectCommentRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectReactionRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectActivityRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}

// rowSteps are d's statements in the global order of M4 design 3.5: the
// work items, then the rows under them.
func rowSteps(d ProjectRowsDeleter) []func(context.Context, RowBatch) ([]uuid.UUID, error) {
	return []func(context.Context, RowBatch) ([]uuid.UUID, error){
		d.DeleteProjectIssueRows, d.DeleteProjectAssigneeRows, d.DeleteProjectLabelRows, d.DeleteProjectRelationRows,
		d.DeleteProjectSubscriberRows, d.DeleteProjectCommentRows, d.DeleteProjectReactionRows, d.DeleteProjectActivityRows,
	}
}

// deleteProjectRows soft-deletes every undeleted work-item-side row of the
// project at at by by, in the transaction ctx carries: table by table in
// the global order, each in batches of limit rows, the next batch after
// the greatest id of the last, until a batch finds none (M4 design 3.6).
// A failing batch comes back as itself, and nothing runs after it.
func deleteProjectRows(ctx context.Context, d ProjectRowsDeleter, limit int, projectID, by uuid.UUID, at time.Time) error {
	for _, step := range rowSteps(d) {
		b := RowBatch{ProjectID: projectID, Limit: limit, By: by, At: at}
		for {
			ids, err := step(ctx, b)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				break
			}
			b.After = slices.MaxFunc(ids, uuid.UUID.Compare)
		}
	}
	return nil
}
