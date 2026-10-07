package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListLabels serves GET /api/v0/projects/{project_id}/labels: data an
// array, never null.
func (h handler) ListLabels(ctx context.Context, req gen.ListLabelsRequestObject) (gen.ListLabelsResponseObject, error) {
	list, err := h.uc.ListLabels.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	out := gen.LabelList{Data: make([]gen.Label, len(list))}
	for i, l := range list {
		out.Data[i] = label(l)
	}
	return gen.ListLabels200JSONResponse(out), nil
}

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

// UpdateLabel serves PATCH /api/v0/labels/{label_id}: the fields given go
// to the use case, a null parent as one given as none.
func (h handler) UpdateLabel(ctx context.Context, req gen.UpdateLabelRequestObject) (gen.UpdateLabelResponseObject, error) {
	p := domain.LabelPatch{Name: req.Body.Name, Color: req.Body.Color, SortOrder: req.Body.SortOrder}
	p.SetParent, p.ParentID = named(req.Body.ParentID)
	l, err := h.uc.UpdateLabel.Execute(ctx, req.LabelID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateLabel200JSONResponse(label(l)), nil
}

// DeleteLabel serves DELETE /api/v0/labels/{label_id}.
func (h handler) DeleteLabel(ctx context.Context, req gen.DeleteLabelRequestObject) (gen.DeleteLabelResponseObject, error) {
	if err := h.uc.DeleteLabel.Execute(ctx, req.LabelID); err != nil {
		return nil, err
	}
	return gen.DeleteLabel204Response{}, nil
}

// label is l as the API shows it: its parent null at the top.
func label(l domain.Label) gen.Label {
	return gen.Label{ID: l.ID, WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID, ParentID: orNull(l.ParentID), Name: l.Name, Color: l.Color,
		SortOrder: l.SortOrder, CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}
