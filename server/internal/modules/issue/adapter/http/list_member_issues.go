package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

func (h handler) ListMemberIssues(ctx context.Context, req gen.ListMemberIssuesRequestObject) (gen.ListMemberIssuesResponseObject, error) {
	page, err := h.uc.ListMemberIssues.Execute(ctx, req.Slug, req.UserID, rawListQuery(ctx))
	if err != nil {
		return nil, err
	}
	grouped := req.Params.GroupBy != nil && req.Params.GroupKey == nil
	return gen.ListMemberIssues200JSONResponse(listPage(page, grouped, false)), nil
}
