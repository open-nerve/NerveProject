package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// InsertComment writes the sanitized content and the creation audit stamp.
func (s *Store) InsertComment(ctx context.Context, w app.CommentWrite) error {
	c := w.Comment
	err := s.queries(ctx).InsertComment(ctx, gen.InsertCommentParams{ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, IssueID: c.IssueID, ActorID: c.ActorID, CommentHtml: c.HTML, CommentStripped: c.Text, By: &w.By, At: w.At})
	if err != nil {
		return fmt.Errorf("insert comment %s: %w", c.ID, err)
	}
	return nil
}

// Comments reads only this issue's live rows in ascending creation order.
func (s *Store) Comments(ctx context.Context, id uuid.UUID, after *domain.CommentCursor, limit int) ([]domain.Comment, error) {
	p := gen.CommentsParams{IssueID: id, PageLimit: int32(limit)}
	if after != nil {
		p.HasAfter, p.AfterAt, p.AfterID = true, after.CreatedAt, after.ID
	}
	rows, err := s.queries(ctx).Comments(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("read comments of work item %s: %w", id, err)
	}
	out := make([]domain.Comment, len(rows))
	for n, r := range rows {
		out[n] = commentOf(r)
	}
	return out, nil
}

// GetComment reads a live comment, without deciding visibility of its issue.
func (s *Store) GetComment(ctx context.Context, id uuid.UUID) (domain.Comment, bool, error) {
	r, err := s.queries(ctx).GetComment(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Comment{}, false, nil
	}
	if err != nil {
		return domain.Comment{}, false, fmt.Errorf("read comment %s: %w", id, err)
	}
	return commentOf(gen.CommentsRow(r)), true, nil
}
func commentOf(r gen.CommentsRow) domain.Comment {
	return domain.Comment{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, ActorID: r.ActorID, HTML: r.CommentHtml, Text: r.CommentStripped, EditedAt: r.EditedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Reactions: []domain.CommentReaction{}}
}

// UpdateComment updates only mutable content and edit audit fields.
func (s *Store) UpdateComment(ctx context.Context, w app.CommentWrite) error {
	err := s.queries(ctx).UpdateComment(ctx, gen.UpdateCommentParams{ID: w.Comment.ID, CommentHtml: w.Comment.HTML, CommentStripped: w.Comment.Text, By: w.By, At: w.At})
	if err != nil {
		return fmt.Errorf("update comment %s: %w", w.Comment.ID, err)
	}
	return nil
}

// DeleteComment preserves existing deletion stamps by updating only live rows.
func (s *Store) DeleteComment(ctx context.Context, d app.CommentDeletion) error {
	err := s.queries(ctx).DeleteComment(ctx, gen.DeleteCommentParams{ID: d.ID, By: d.By, At: d.At})
	if err != nil {
		return fmt.Errorf("delete comment %s: %w", d.ID, err)
	}
	return nil
}
