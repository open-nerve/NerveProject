package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// SearchIssues preserves the complete raw query for the search use case.
func (h handler) SearchIssues(ctx context.Context, req gen.SearchIssuesRequestObject) (gen.SearchIssuesResponseObject, error) {
	rows, err := h.uc.SearchIssues.Execute(ctx, req.Slug, rawListQuery(ctx))
	if err != nil {
		return nil, err
	}
	return gen.SearchIssues200JSONResponse(issueCollection(rows)), nil
}
