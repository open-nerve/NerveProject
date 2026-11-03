package httpadapter

import (
	"context"
	"net/http"
	"net/url"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type GetAttachmentsUseCase interface {
	Execute(context.Context, uuid.UUID, *int, *string) (app.AttachmentPage, error)
}

func (h handler) GetIssueAttachments(ctx context.Context, req gen.GetIssueAttachmentsRequestObject) (gen.GetIssueAttachmentsResponseObject, error) {
	p, err := h.uc.GetIssueAttachments.Execute(ctx, req.IssueID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	return gen.GetIssueAttachments200JSONResponse(attachmentPage(p)), nil
}
func (h handler) GetCommentAttachments(ctx context.Context, req gen.GetCommentAttachmentsRequestObject) (gen.GetCommentAttachmentsResponseObject, error) {
	p, err := h.uc.GetCommentAttachments.Execute(ctx, req.CommentID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	return gen.GetCommentAttachments200JSONResponse(attachmentPage(p)), nil
}
func (h handler) GetDraftAttachments(ctx context.Context, req gen.GetDraftAttachmentsRequestObject) (gen.GetDraftAttachmentsResponseObject, error) {
	p, err := h.uc.GetDraftAttachments.Execute(ctx, req.DraftID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	return gen.GetDraftAttachments200JSONResponse(attachmentPage(p)), nil
}
func attachmentPage(p app.AttachmentPage) gen.AttachmentPage {
	out := gen.AttachmentPage{Data: make([]gen.Attachment, len(p.Data))}
	var cursor *string
	if p.NextCursor != "" {
		cursor = &p.NextCursor
	}
	out.NextCursor = orNull(cursor)
	for i, a := range p.Data {
		out.Data[i] = gen.Attachment{ID: a.ID, Name: a.Name, ContentType: a.ContentType, Size: a.Size, CreatedAt: a.CreatedAt, CreatedByID: orNull(a.CreatedByID)}
	}
	return out
}
func withAttachmentQuery(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
	if operationID != "GetIssueAttachments" && operationID != "GetCommentAttachments" && operationID != "GetDraftAttachments" {
		return next
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			return nil, shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The request query could not be decoded.")
		}
		for key, values := range q {
			if (key != "limit" && key != "cursor") || len(values) != 1 {
				return nil, shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The attachment query has an unknown or repeated parameter.")
			}
		}
		return next(ctx, w, r, request)
	}
}
