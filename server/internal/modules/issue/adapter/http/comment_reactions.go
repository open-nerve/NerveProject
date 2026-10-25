package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// AddCommentReaction serves POST /api/v0/issue-comments/{comment_id}/reactions.
func (h handler) AddCommentReaction(ctx context.Context, req gen.AddCommentReactionRequestObject) (gen.AddCommentReactionResponseObject, error) {
	r, err := h.uc.AddCommentReaction.Execute(ctx, req.CommentID, req.Body.Reaction)
	if err != nil {
		return nil, err
	}
	return gen.AddCommentReaction201JSONResponse(commentReactionDetail(r)), nil
}

// RemoveCommentReaction serves DELETE /api/v0/issue-comments/{comment_id}/reactions/{reaction}.
func (h handler) RemoveCommentReaction(ctx context.Context, req gen.RemoveCommentReactionRequestObject) (gen.RemoveCommentReactionResponseObject, error) {
	if err := h.uc.RemoveCommentReaction.Execute(ctx, req.CommentID, req.Reaction); err != nil {
		return nil, err
	}
	return gen.RemoveCommentReaction204Response{}, nil
}

func commentReactionDetail(r domain.CommentReaction) gen.CommentReaction {
	return gen.CommentReaction{ID: r.ID, CommentID: r.CommentID, ActorID: r.ActorID, Reaction: r.Reaction, CreatedAt: r.CreatedAt}
}
