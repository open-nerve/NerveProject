package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// ListProjectViews serves GET /api/v0/projects/{project_id}/views.
func (h handler) ListProjectViews(ctx context.Context, req gen.ListProjectViewsRequestObject) (gen.ListProjectViewsResponseObject, error) {
	list, err := h.uc.ListProjectViews.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.ListProjectViews200JSONResponse(viewList(list)), nil
}

// CreateProjectView serves POST /api/v0/projects/{project_id}/views: the
// body's values go to the use case as given (newView), which checks them.
func (h handler) CreateProjectView(ctx context.Context, req gen.CreateProjectViewRequestObject) (gen.CreateProjectViewResponseObject, error) {
	v, err := h.uc.CreateProjectView.Execute(ctx, req.ProjectID, newView(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.CreateProjectView201JSONResponse(viewOf(v)), nil
}

// ListWorkspaceViews serves GET /api/v0/workspaces/{slug}/views.
func (h handler) ListWorkspaceViews(ctx context.Context, req gen.ListWorkspaceViewsRequestObject) (gen.ListWorkspaceViewsResponseObject, error) {
	list, err := h.uc.ListWorkspaceViews.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.ListWorkspaceViews200JSONResponse(viewList(list)), nil
}

// CreateWorkspaceView serves POST /api/v0/workspaces/{slug}/views, as
// CreateProjectView.
func (h handler) CreateWorkspaceView(ctx context.Context, req gen.CreateWorkspaceViewRequestObject) (gen.CreateWorkspaceViewResponseObject, error) {
	v, err := h.uc.CreateWorkspaceView.Execute(ctx, req.Slug, newView(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.CreateWorkspaceView201JSONResponse(viewOf(v)), nil
}

// GetView serves GET /api/v0/views/{view_id}.
func (h handler) GetView(ctx context.Context, req gen.GetViewRequestObject) (gen.GetViewResponseObject, error) {
	v, err := h.uc.GetView.Execute(ctx, req.ViewID)
	if err != nil {
		return nil, err
	}
	return gen.GetView200JSONResponse(viewOf(v)), nil
}

// UpdateView serves PATCH /api/v0/views/{view_id}: the fields the body
// gives go to the use case, each whole, those it leaves out nil (patchOf).
func (h handler) UpdateView(ctx context.Context, req gen.UpdateViewRequestObject) (gen.UpdateViewResponseObject, error) {
	v, err := h.uc.UpdateView.Execute(ctx, req.ViewID, patchOf(*req.Body))
	if err != nil {
		return nil, err
	}
	return gen.UpdateView200JSONResponse(viewOf(v)), nil
}

// DeleteView serves DELETE /api/v0/views/{view_id}.
func (h handler) DeleteView(ctx context.Context, req gen.DeleteViewRequestObject) (gen.DeleteViewResponseObject, error) {
	if err := h.uc.DeleteView.Execute(ctx, req.ViewID); err != nil {
		return nil, err
	}
	return gen.DeleteView204Response{}, nil
}

// viewList is list as the API shows it: [] for none, never null.
func viewList(list []domain.View) gen.ViewList {
	out := gen.ViewList{Data: make([]gen.View, len(list))}
	for i, v := range list {
		out.Data[i] = viewOf(v)
	}
	return out
}
