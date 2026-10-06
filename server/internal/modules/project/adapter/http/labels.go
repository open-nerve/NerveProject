package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// CreateLabel serves POST /api/v0/projects/{project_id}/labels: the color
// empty and the parent nil when not given.
func (h handler) CreateLabel(ctx context.Context, req gen.CreateLabelRequestObject) (gen.CreateLabelResponseObject, error) {
	in := domain.LabelCreate{Name: req.Body.Name, ParentID: req.Body.ParentID}
	if req.Body.Color != nil {
		in.Color = *req.Body.Color
	}
	l, err := h.uc.CreateLabel.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateLabel201JSONResponse(label(l)), nil
}

// label is l as the API shows it: its parent null at the top.
func label(l domain.Label) gen.Label {
	return gen.Label{ID: l.ID, WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID, ParentID: orNull(l.ParentID), Name: l.Name, Color: l.Color,
		SortOrder: l.SortOrder, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}
