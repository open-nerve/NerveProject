package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type fakeComments struct {
	*fakeRelations
	inserted   *app.CommentWrite
	subscribed *app.LinkRows
	listed     []domain.Comment
	limit      int
	after      *domain.CommentCursor
}

func (f *fakeComments) InsertComment(ctx context.Context, w app.CommentWrite) error {
	if err := f.log.add(ctx, "InsertComment", w.Comment.IssueID); err != nil {
		return err
	}
	f.inserted = &w
	return nil
}
func (f *fakeComments) InsertSubscribers(ctx context.Context, r app.LinkRows) error {
	if err := f.log.add(ctx, "InsertSubscribers", r.Targets); err != nil {
		return err
	}
	f.subscribed = &r
	return nil
}
func (f *fakeComments) Comments(ctx context.Context, id uuid.UUID, after *domain.CommentCursor, limit int) ([]domain.Comment, error) {
	if err := f.log.add(ctx, "Comments", id); err != nil {
		return nil, err
	}
	f.limit, f.after = limit, after
	return f.listed, nil
}
func (f *fakeProject) MentionMembers(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]app.MemberVisibility, error) {
	if err := f.log.add(ctx, "MentionMembers", id, ids); err != nil {
		return nil, err
	}
	return f.mentionVisibility, nil
}
