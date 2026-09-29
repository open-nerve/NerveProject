package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// ListWorkspaces serves GET /api/v0/workspaces.
func (h handler) ListWorkspaces(ctx context.Context, _ gen.ListWorkspacesRequestObject) (gen.ListWorkspacesResponseObject, error) {
	list, err := h.uc.ListWorkspaces.Execute(ctx)
	if err != nil {
		return nil, err
	}
	out := gen.ListWorkspaces200JSONResponse{Data: make([]gen.Workspace, len(list))}
	for i, w := range list {
		out.Data[i] = workspace(w)
	}
	return out, nil
}

// CreateWorkspace serves POST /api/v0/workspaces.
func (h handler) CreateWorkspace(ctx context.Context, req gen.CreateWorkspaceRequestObject) (gen.CreateWorkspaceResponseObject, error) {
	in := domain.NewWorkspace{Name: req.Body.Name, Slug: req.Body.Slug, Timezone: req.Body.Timezone}
	if size := req.Body.OrganizationSize; size != nil {
		s := string(*size)
		in.OrganizationSize = &s
	}
	w, err := h.uc.CreateWorkspace.Execute(ctx, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateWorkspace201JSONResponse(workspace(w)), nil
}

// GetWorkspace serves GET /api/v0/workspaces/{slug}.
func (h handler) GetWorkspace(ctx context.Context, req gen.GetWorkspaceRequestObject) (gen.GetWorkspaceResponseObject, error) {
	w, err := h.uc.GetWorkspace.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspace200JSONResponse(workspace(w)), nil
}

// UpdateWorkspace serves PATCH /api/v0/workspaces/{slug}.
func (h handler) UpdateWorkspace(ctx context.Context, req gen.UpdateWorkspaceRequestObject) (gen.UpdateWorkspaceResponseObject, error) {
	p := domain.WorkspacePatch{Name: req.Body.Name, Timezone: req.Body.Timezone}
	if size := req.Body.OrganizationSize; size != nil {
		s := string(*size)
		p.OrganizationSize = &s
	}
	w, err := h.uc.UpdateWorkspace.Execute(ctx, req.Slug, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspace200JSONResponse(workspace(w)), nil
}

// CheckWorkspaceSlug serves GET /api/v0/workspace-slugs/{slug}.
func (h handler) CheckWorkspaceSlug(ctx context.Context, req gen.CheckWorkspaceSlugRequestObject) (gen.CheckWorkspaceSlugResponseObject, error) {
	reason, err := h.uc.CheckSlug.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	if reason == "" {
		return gen.CheckWorkspaceSlug200JSONResponse{Available: true}, nil
	}
	r := gen.SlugAvailabilityReason(reason)
	return gen.CheckWorkspaceSlug200JSONResponse{Available: false, Reason: &r}, nil
}

// workspace is w as the API shows it.
func workspace(w domain.Workspace) gen.Workspace {
	size := nullable.NewNullNullable[string]()
	if w.OrganizationSize != nil {
		size = nullable.NewNullableWithValue(*w.OrganizationSize)
	}
	return gen.Workspace{
		ID:               w.ID,
		Name:             w.Name,
		Slug:             w.Slug,
		OrganizationSize: size,
		Timezone:         w.Timezone,
		// Required and null until M5. The zero Nullable is "unspecified" and
		// would marshal as "": set null explicitly.
		LogoURL:      nullable.NewNullNullable[string](),
		Role:         gen.WorkspaceRole(w.Role),
		TotalMembers: w.TotalMembers,
		CreatedAt:    w.CreatedAt,
		UpdatedAt:    w.UpdatedAt,
	}
}
