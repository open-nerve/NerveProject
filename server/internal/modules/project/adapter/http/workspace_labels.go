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
	out := gen.LabelList{Data: make([]gen.Label, len(list))}
	for i, l := range list {
		out.Data[i] = label(l)
	}
	return gen.ListWorkspaceLabels200JSONResponse(out), nil
}
