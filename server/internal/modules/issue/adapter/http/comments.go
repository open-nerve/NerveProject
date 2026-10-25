package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssueComment serves POST /api/v0/issues/{issue_id}/comments.
func (h handler) CreateIssueComment(ctx context.Context, req gen.CreateIssueCommentRequestObject) (gen.CreateIssueCommentResponseObject, error) {
	c, err := h.uc.CreateIssueComment.Execute(ctx, req.IssueID, req.Body.CommentHTML)
	if err != nil {
		return nil, err
	}
	return gen.CreateIssueComment201JSONResponse(commentDetail(c)), nil
}

// ListIssueComments serves GET /api/v0/issues/{issue_id}/comments.
func (h handler) ListIssueComments(ctx context.Context, req gen.ListIssueCommentsRequestObject) (gen.ListIssueCommentsResponseObject, error) {
	page, err := h.uc.ListIssueComments.Execute(ctx, req.IssueID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	data := make([]gen.IssueComment, len(page.Data))
	for n, c := range page.Data {
		data[n] = commentDetail(c)
	}
	var cursor *string
	if page.NextCursor != "" {
		cursor = &page.NextCursor
	}
	return gen.ListIssueComments200JSONResponse{Data: data, NextCursor: orNull(cursor)}, nil
}
func commentDetail(c domain.Comment) gen.IssueComment {
	reactions := make([]gen.CommentReaction, len(c.Reactions))
	for n, r := range c.Reactions {
		reactions[n] = gen.CommentReaction{ID: r.ID, CommentID: r.CommentID, ActorID: r.ActorID, Reaction: r.Reaction, CreatedAt: r.CreatedAt}
	}
	return gen.IssueComment{ID: c.ID, IssueID: c.IssueID, ActorID: c.ActorID, CommentHTML: c.HTML, EditedAt: orNull(c.EditedAt), CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, Reactions: reactions}
}
