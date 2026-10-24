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

// InsertCommentReaction writes a reaction and its audit stamp.
func (s *Store) InsertCommentReaction(ctx context.Context, w app.CommentReactionWrite) error {
	r := w.Reaction
	if err := s.queries(ctx).InsertCommentReaction(ctx, gen.InsertCommentReactionParams{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, CommentID: r.CommentID, ActorID: r.ActorID, Reaction: r.Reaction, By: &w.By, At: w.At}); err != nil {
		return fmt.Errorf("insert reaction %s: %w", r.ID, err)
	}
	return nil
}

// CommentReaction reads only the named actor's live reaction on this comment.
func (s *Store) CommentReaction(ctx context.Context, k app.ReactionKey) (domain.CommentReaction, bool, error) {
	r, err := s.queries(ctx).CommentReaction(ctx, gen.CommentReactionParams{CommentID: k.ResourceID, ActorID: k.ActorID, Reaction: k.Reaction})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.CommentReaction{}, false, nil
	}
	if err != nil {
		return domain.CommentReaction{}, false, fmt.Errorf("read comment reaction: %w", err)
	}
	return commentReactionOf(gen.CommentReactionsRow(r)), true, nil
}

// CommentReactions reads live rows in ascending (created_at,id) order.
func (s *Store) CommentReactions(ctx context.Context, ids []uuid.UUID) ([]domain.CommentReaction, error) {
	rows, err := s.queries(ctx).CommentReactions(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read reactions of %v: %w", ids, err)
	}
	out := make([]domain.CommentReaction, len(rows))
	for n, r := range rows {
		out[n] = commentReactionOf(r)
	}
	return out, nil
}
func commentReactionOf(r gen.CommentReactionsRow) domain.CommentReaction {
	return domain.CommentReaction{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, CommentID: r.CommentID, ActorID: r.ActorID, Reaction: r.Reaction, CreatedAt: r.CreatedAt}
}

// DeleteCommentReaction preserves every other actor, comment and prior deletion.
func (s *Store) DeleteCommentReaction(ctx context.Context, d app.ReactionDeletion) error {
	if err := s.queries(ctx).DeleteCommentReaction(ctx, gen.DeleteCommentReactionParams{CommentID: d.Key.ResourceID, ActorID: d.Key.ActorID, Reaction: d.Key.Reaction, By: d.By, At: d.At}); err != nil {
		return fmt.Errorf("delete comment reaction: %w", err)
	}
	return nil
}

// DeleteCommentReactionRows soft-deletes all live reactions of the deleted comment.
func (s *Store) DeleteCommentReactionRows(ctx context.Context, d app.CommentDeletion) error {
	if err := s.queries(ctx).DeleteCommentReactionRows(ctx, gen.DeleteCommentReactionRowsParams{CommentID: d.ID, At: d.At, By: d.By}); err != nil {
		return fmt.Errorf("delete comment reactions: %w", err)
	}
	return nil
}
