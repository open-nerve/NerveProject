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
	methods := map[RowStep]func(context.Context, RowBatch) (BatchProgress, error){
		"draft_issues": rowProgress(d.DeleteProjectDraftRows), "issues": rowProgress(d.DeleteProjectIssueRows),
		"issue_assignees": rowProgress(d.DeleteProjectAssigneeRows), "issue_labels": rowProgress(d.DeleteProjectLabelRows),
		"issue_relations": relations, "issue_links": rowProgress(d.DeleteProjectLinkRows),
		"issue_subscribers": rowProgress(d.DeleteProjectSubscriberRows), "issue_mentions": rowProgress(d.DeleteProjectMentionRows),
		"issue_comments": rowProgress(d.DeleteProjectCommentRows), "issue_reactions": rowProgress(d.DeleteProjectReactionRows),
		"comment_reactions": rowProgress(d.DeleteProjectCommentReactionRows), "issue_activities": rowProgress(d.DeleteProjectActivityRows),
		"issue_description_versions": rowProgress(d.DeleteProjectDescriptionVersionRows),
	}
	var steps []func(context.Context, RowBatch) (BatchProgress, error)
	for _, name := range DeletionRowSteps() {
		steps = append(steps, methods[name])
	}
	return steps
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
