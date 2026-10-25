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

// InsertIssueReaction writes a reaction and its audit stamp.
func (s *Store) InsertIssueReaction(ctx context.Context, w app.IssueReactionWrite) error {
	r := w.Reaction
	if err := s.queries(ctx).InsertIssueReaction(ctx, gen.InsertIssueReactionParams{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, ActorID: r.ActorID, Reaction: r.Reaction, By: &w.By, At: w.At}); err != nil {
		return fmt.Errorf("insert reaction %s: %w", r.ID, err)
	}
	return nil
}

// IssueReaction reads only the named actor's live reaction on this issue.
func (s *Store) IssueReaction(ctx context.Context, k app.ReactionKey) (domain.IssueReaction, bool, error) {
	r, err := s.queries(ctx).IssueReaction(ctx, gen.IssueReactionParams{IssueID: k.ResourceID, ActorID: k.ActorID, Reaction: k.Reaction})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.IssueReaction{}, false, nil
	}
	if err != nil {
		return domain.IssueReaction{}, false, fmt.Errorf("read issue reaction: %w", err)
	}
	return reactionOf(gen.IssueReactionsRow(r)), true, nil
}

// IssueReactions reads live rows in ascending (created_at,id) order.
func (s *Store) IssueReactions(ctx context.Context, id uuid.UUID) ([]domain.IssueReaction, error) {
	rows, err := s.queries(ctx).IssueReactions(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read reactions of %s: %w", id, err)
	}
	out := make([]domain.IssueReaction, len(rows))
	for n, r := range rows {
		out[n] = reactionOf(r)
	}
	return out, nil
}
func reactionOf(r gen.IssueReactionsRow) domain.IssueReaction {
	return domain.IssueReaction{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, ActorID: r.ActorID, Reaction: r.Reaction, CreatedAt: r.CreatedAt}
}

// DeleteIssueReaction preserves every other actor, issue and prior deletion.
func (s *Store) DeleteIssueReaction(ctx context.Context, d app.ReactionDeletion) error {
	if err := s.queries(ctx).DeleteIssueReaction(ctx, gen.DeleteIssueReactionParams{IssueID: d.Key.ResourceID, ActorID: d.Key.ActorID, Reaction: d.Key.Reaction, By: d.By, At: d.At}); err != nil {
		return fmt.Errorf("delete issue reaction: %w", err)
	}
	return nil
}
