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

// WorkspaceDraftBatch selects undeleted drafts without a project in a workspace,
// in the caller's transaction and with the same keyset and audit semantics as RowBatch.
type WorkspaceDraftBatch struct {
	WorkspaceID, After, By uuid.UUID
	Limit                  int
	At                     time.Time
}

// ProjectRowsDeleter soft-deletes a deleted project's rows of one table a
// batch at a time, in the transaction ctx carries; each method locks the
// batch's rows FOR NO KEY UPDATE in id order before it writes them
// (convention 5) and returns the ids it deleted.
type ProjectRowsDeleter interface {
	DeleteProjectDraftRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteWorkspaceDraftRows(ctx context.Context, b WorkspaceDraftBatch) ([]uuid.UUID, error)
	DeleteProjectIssueRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectAssigneeRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectLabelRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	// DeleteProjectRelationRows' batch is of the relations whose either
	// end is a work item of the project.
	DeleteProjectRelationRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectLinkRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectSubscriberRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectMentionRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectDescriptionVersionRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectCommentRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectReactionRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectCommentReactionRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
	DeleteProjectActivityRows(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}

// rowSteps are d's statements in the global order of M4 design 3.5: the
// drafts, work items, then the rows under them.
func rowSteps(d ProjectRowsDeleter, relations func(context.Context, RowBatch) (BatchProgress, error)) []func(context.Context, RowBatch) (BatchProgress, error) {
	return []func(context.Context, RowBatch) (BatchProgress, error){
		rowProgress(d.DeleteProjectDraftRows), rowProgress(d.DeleteProjectIssueRows), rowProgress(d.DeleteProjectAssigneeRows), rowProgress(d.DeleteProjectLabelRows), relations, rowProgress(d.DeleteProjectLinkRows),
		rowProgress(d.DeleteProjectSubscriberRows), rowProgress(d.DeleteProjectMentionRows), rowProgress(d.DeleteProjectCommentRows), rowProgress(d.DeleteProjectReactionRows), rowProgress(d.DeleteProjectCommentReactionRows), rowProgress(d.DeleteProjectActivityRows), rowProgress(d.DeleteProjectDescriptionVersionRows),
	}
}

func rowProgress(step func(context.Context, RowBatch) ([]uuid.UUID, error)) func(context.Context, RowBatch) (BatchProgress, error) {
	return func(ctx context.Context, b RowBatch) (BatchProgress, error) {
		ids, err := step(ctx, b)
		if err != nil {
			return BatchProgress{}, err
		}
		progress := BatchProgress{Candidates: len(ids), Written: len(ids)}
		if len(ids) > 0 {
			progress.After = slices.MaxFunc(ids, uuid.UUID.Compare)
		}
		return progress, nil
	}
}

// deleteRowBatches is the shared candidate scan: ownership can skip every row
// in a batch, but only an empty candidate batch ends the table.
func deleteRowBatches(ctx context.Context, step func(context.Context, RowBatch) (BatchProgress, error), b RowBatch) error {
	for {
		progress, err := step(ctx, b)
		if err != nil {
			return err
		}
		if progress.Candidates == 0 {
			return nil
		}
		b.After = progress.After
	}
}

// deleteProjectRows soft-deletes every undeleted work-item-side row of the
// project at at by by, in the transaction ctx carries: table by table in
// the global order, each in batches of limit rows, the next batch after
// the greatest id of the last, until a batch finds none (M4 design 3.6).
// A failing batch comes back as itself, and nothing runs after it.
func deleteProjectRows(ctx context.Context, d ProjectRowsDeleter, limit int, projectID, by uuid.UUID, at time.Time) error {
	// The synchronous Cascade keeps its original relation statement until T10.
	for _, step := range rowSteps(d, rowProgress(d.DeleteProjectRelationRows)) {
		if err := deleteRowBatches(ctx, step, RowBatch{ProjectID: projectID, Limit: limit, By: by, At: at}); err != nil {
			return err
		}
	}
	return nil
}
