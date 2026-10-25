package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// UpdateIssueComment serves PATCH /api/v0/issue-comments/{comment_id}.
func (h handler) UpdateIssueComment(ctx context.Context, req gen.UpdateIssueCommentRequestObject) (gen.UpdateIssueCommentResponseObject, error) {
	c, err := h.uc.UpdateIssueComment.Execute(ctx, req.CommentID, req.Body.CommentHTML)
	if err != nil {
		return nil, err
	}
	return gen.UpdateIssueComment200JSONResponse(commentDetail(c)), nil
}

// DeleteIssueComment serves DELETE /api/v0/issue-comments/{comment_id}.
func (h handler) DeleteIssueComment(ctx context.Context, req gen.DeleteIssueCommentRequestObject) (gen.DeleteIssueCommentResponseObject, error) {
	if err := h.uc.DeleteIssueComment.Execute(ctx, req.CommentID); err != nil {
		return nil, err
	}
	return gen.DeleteIssueComment204Response{}, nil
}
