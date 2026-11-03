package httpadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/bodyshape"
)

// CreateIssueComment serves POST /api/v0/issues/{issue_id}/comments.
func (h handler) CreateIssueComment(ctx context.Context, req gen.CreateIssueCommentRequestObject) (gen.CreateIssueCommentResponseObject, error) {
	in := domain.NewComment{HTML: req.Body.CommentHTML}
	if req.Body.AttachmentIds != nil {
		// Keep the original spelling until this boundary; UUID.UnmarshalJSON
		// would otherwise silently normalize upper-case and compact values.
		in.AttachmentIDs = make([]uuid.UUID, len(*req.Body.AttachmentIds))
		for n, raw := range *req.Body.AttachmentIds {
			id, err := uuid.Parse(raw)
			if err != nil || id.String() != raw {
				return nil, &bodyshape.Error{Fields: []bodyshape.FieldError{{Field: fmt.Sprintf("attachment_ids[%d]", n), Code: "invalid_format"}}}
			}
			in.AttachmentIDs[n] = id
		}
	}
	c, err := h.uc.CreateIssueComment.Execute(ctx, req.IssueID, in)
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
