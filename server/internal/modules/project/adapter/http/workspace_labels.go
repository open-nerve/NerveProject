package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
)

// ListWorkspaceLabels serves GET /api/v0/workspaces/{slug}/labels: each
// label as listLabels shows it, data an array, never null.
func (h handler) ListWorkspaceLabels(ctx context.Context, req gen.ListWorkspaceLabelsRequestObject) (gen.ListWorkspaceLabelsResponseObject, error) {
	list, err := h.uc.ListWorkspaceLabels.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceLabels200JSONResponse(labelList(list)), nil
}
