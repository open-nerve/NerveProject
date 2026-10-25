package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type fakeReactions struct {
	*fakeRelations
	rows    map[app.ReactionKey]domain.IssueReaction
	added   *app.IssueReactionWrite
	removed *app.ReactionDeletion
	listed  []domain.IssueReaction
}

func (f *fakeReactions) IssueReaction(ctx context.Context, k app.ReactionKey) (domain.IssueReaction, bool, error) {
	if e := f.log.add(ctx, "IssueReaction", k); e != nil {
		return domain.IssueReaction{}, false, e
	}
	r, ok := f.rows[k]
	return r, ok, nil
}
func (f *fakeReactions) InsertIssueReaction(ctx context.Context, w app.IssueReactionWrite) error {
	if e := f.log.add(ctx, "InsertIssueReaction", w.Reaction.IssueID); e != nil {
		return e
	}
	f.added = &w
	f.rows[app.ReactionKey{ResourceID: w.Reaction.IssueID, ActorID: w.Reaction.ActorID, Reaction: w.Reaction.Reaction}] = w.Reaction
	return nil
}
func (f *fakeReactions) DeleteIssueReaction(ctx context.Context, d app.ReactionDeletion) error {
	if e := f.log.add(ctx, "DeleteIssueReaction", d.Key); e != nil {
		return e
	}
	f.removed = &d
	delete(f.rows, d.Key)
	return nil
}
func (f *fakeReactions) IssueReactions(ctx context.Context, id uuid.UUID) ([]domain.IssueReaction, error) {
	if e := f.log.add(ctx, "IssueReactions", id); e != nil {
		return nil, e
	}
	return f.listed, nil
}
