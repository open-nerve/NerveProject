package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssueLink serves POST /api/v0/issues/{issue_id}/links.
func (h handler) CreateIssueLink(ctx context.Context, req gen.CreateIssueLinkRequestObject) (gen.CreateIssueLinkResponseObject, error) {
	_, title := given(req.Body.Title)
	link, err := h.uc.CreateIssueLink.Execute(ctx, req.IssueID, domain.NewLink{URL: req.Body.URL, Title: title})
	if err != nil {
		return nil, err
	}
	return gen.CreateIssueLink201JSONResponse(linkDetail(link)), nil
}

// ListIssueLinks serves GET /api/v0/issues/{issue_id}/links.
func (h handler) ListIssueLinks(ctx context.Context, req gen.ListIssueLinksRequestObject) (gen.ListIssueLinksResponseObject, error) {
	links, err := h.uc.ListIssueLinks.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	data := make([]gen.IssueLink, len(links))
	for n, link := range links {
		data[n] = linkDetail(link)
	}
	return gen.ListIssueLinks200JSONResponse{Data: data}, nil
}
func linkDetail(link domain.Link) gen.IssueLink {
	m := gen.IssueLinkMetadata{}
	if value, ok := link.Metadata["title"]; ok {
		m.Title = orNull(value)
	}
	if value, ok := link.Metadata["favicon"]; ok {
		m.Favicon = orNull(value)
	}
	if value, ok := link.Metadata["url"]; ok {
		m.URL = orNull(value)
	}
	if value, ok := link.Metadata["favicon_url"]; ok {
		m.FaviconURL = orNull(value)
	}
	if value, ok := link.Metadata["error"]; ok {
		m.Error = orNull(value)
	}
	return gen.IssueLink{ID: link.ID, IssueID: link.IssueID, URL: link.URL, Title: orNull(link.Title), Metadata: m, CreatedByID: orNull(link.CreatedByID), CreatedAt: link.CreatedAt, UpdatedAt: link.UpdatedAt}
}
