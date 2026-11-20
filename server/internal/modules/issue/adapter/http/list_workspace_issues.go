package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// ListWorkspaceIssues answers the workspace's list, always flat (M7 P3a
// spec 2.3): the use case gets the raw query (withListQuery).
func (h handler) ListWorkspaceIssues(ctx context.Context, req gen.ListWorkspaceIssuesRequestObject) (gen.ListWorkspaceIssuesResponseObject, error) {
	page, err := h.uc.ListWorkspaceIssues.Execute(ctx, req.Slug, rawListQuery(ctx))
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceIssues200JSONResponse(listPage(page, false, false)), nil
}
