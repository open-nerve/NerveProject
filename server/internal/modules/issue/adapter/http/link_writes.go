package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// UpdateIssueLink serves PATCH /api/v0/issue-links/{link_id}.
func (h handler) UpdateIssueLink(ctx context.Context, req gen.UpdateIssueLinkRequestObject) (gen.UpdateIssueLinkResponseObject, error) {
	setTitle, title := given(req.Body.Title)
	link, err := h.uc.UpdateIssueLink.Execute(ctx, req.LinkID, domain.LinkPatch{URL: req.Body.URL, SetTitle: setTitle, Title: title})
	if err != nil {
		return nil, err
	}
	return gen.UpdateIssueLink200JSONResponse(linkDetail(link)), nil
}

// DeleteIssueLink serves DELETE /api/v0/issue-links/{link_id}.
func (h handler) DeleteIssueLink(ctx context.Context, req gen.DeleteIssueLinkRequestObject) (gen.DeleteIssueLinkResponseObject, error) {
	if err := h.uc.DeleteIssueLink.Execute(ctx, req.LinkID); err != nil {
		return nil, err
	}
	return gen.DeleteIssueLink204Response{}, nil
}
