package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The batches of a deleted project's work-item side (app.ProjectRowsDeleter,
// delete_rows.sql): each soft-deletes the batch's rows of one table and
// returns their ids.

// batch is b as the statements take it; their parameters are of one shape.
func batch(b app.RowBatch) gen.DeleteProjectIssueRowsParams {
	return gen.DeleteProjectIssueRowsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)}
}

// deleted returns the batch's ids, or its failure named by what.
func deleted(ids []uuid.UUID, err error, what string) ([]uuid.UUID, error) {
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's %s: %w", what, err)
	}
	return ids, nil
}

// DeleteProjectIssueRows soft-deletes a batch of the project's work items,
// archived ones too.
func (s *Store) DeleteProjectIssueRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectIssueRows(ctx, batch(b))
	return deleted(ids, err, "work items")
}

// DeleteProjectAssigneeRows soft-deletes a batch of the project's work
// items' assignees.
func (s *Store) DeleteProjectAssigneeRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectAssigneeRows(ctx, gen.DeleteProjectAssigneeRowsParams(batch(b)))
	return deleted(ids, err, "assignees")
}

// DeleteProjectLabelRows soft-deletes a batch of the labels on the
// project's work items.
func (s *Store) DeleteProjectLabelRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectLabelRows(ctx, gen.DeleteProjectLabelRowsParams(batch(b)))
	return deleted(ids, err, "work items' labels")
}

// DeleteProjectRelationRows soft-deletes a batch of the relations of the
// project's work items, both ways.
func (s *Store) DeleteProjectRelationRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectRelationRows(ctx, gen.DeleteProjectRelationRowsParams(batch(b)))
	return deleted(ids, err, "work items' relations")
}

// DeleteProjectLinkRows soft-deletes a batch of the project's links.
func (s *Store) DeleteProjectLinkRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectLinkRows(ctx, gen.DeleteProjectLinkRowsParams(batch(b)))
	return deleted(ids, err, "links")
}

// DeleteProjectSubscriberRows soft-deletes a batch of the project's work
// items' subscribers.
func (s *Store) DeleteProjectSubscriberRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectSubscriberRows(ctx, gen.DeleteProjectSubscriberRowsParams(batch(b)))
	return deleted(ids, err, "subscribers")
}

// DeleteProjectActivityRows soft-deletes a batch of the project's work
// items' activity.
func (s *Store) DeleteProjectActivityRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectActivityRows(ctx, gen.DeleteProjectActivityRowsParams(batch(b)))
	return deleted(ids, err, "activity")
}

// DeleteProjectCommentRows soft-deletes comment rows in the existing deletion sequence.
func (s *Store) DeleteProjectCommentRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectCommentRows(ctx, gen.DeleteProjectCommentRowsParams(batch(b)))
	return deleted(ids, err, "comments")
}

// DeleteProjectReactionRows soft-deletes reaction rows in the existing deletion sequence.
func (s *Store) DeleteProjectReactionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectReactionRows(ctx, gen.DeleteProjectReactionRowsParams(batch(b)))
	return deleted(ids, err, "reactions")
}

// DeleteProjectCommentReactionRows soft-deletes comment reaction rows in the existing deletion sequence.
func (s *Store) DeleteProjectCommentReactionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectCommentReactionRows(ctx, gen.DeleteProjectCommentReactionRowsParams(batch(b)))
	return deleted(ids, err, "comment reactions")
}

// DeleteProjectMentionRows soft deletes one keyset batch of live project rows.
func (s *Store) DeleteProjectMentionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectMentionRows(ctx, gen.DeleteProjectMentionRowsParams(batch(b)))
	return deleted(ids, err, "Mention")
}

// DeleteProjectDescriptionVersionRows soft deletes one keyset batch of live project rows.
func (s *Store) DeleteProjectDescriptionVersionRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectDescriptionVersionRows(ctx, gen.DeleteProjectDescriptionVersionRowsParams(batch(b)))
	return deleted(ids, err, "DescriptionVersion")
}

// DeleteProjectDraftRows soft-deletes one keyset batch of live project drafts.
func (s *Store) DeleteProjectDraftRows(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteProjectDraftRows(ctx, gen.DeleteProjectDraftRowsParams{At: b.At, DeletedBy: b.By, ProjectID: &b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	return deleted(ids, err, "drafts")
}

// DeleteWorkspaceDraftRows soft-deletes live drafts without a project.
func (s *Store) DeleteWorkspaceDraftRows(ctx context.Context, b app.WorkspaceDraftBatch) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).DeleteWorkspaceDraftRows(ctx, gen.DeleteWorkspaceDraftRowsParams{
		At: b.At, DeletedBy: b.By, WorkspaceID: b.WorkspaceID, After: b.After, Batch: int32(b.Limit),
	})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of workspace drafts: %w", err)
	}
	return ids, nil
}
