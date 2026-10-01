package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// ListProjects serves GET /api/v0/workspaces/{slug}/projects.
func (h handler) ListProjects(ctx context.Context, req gen.ListProjectsRequestObject) (gen.ListProjectsResponseObject, error) {
	archived := req.Params.Archived != nil && *req.Params.Archived
	list, err := h.uc.ListProjects.Execute(ctx, req.Slug, archived)
	if err != nil {
		return nil, err
	}
	out := gen.ListProjects200JSONResponse{Data: make([]gen.Project, len(list))}
	for i, p := range list {
		out.Data[i] = project(p)
	}
	return out, nil
}

// CreateProject serves POST /api/v0/workspaces/{slug}/projects.
func (h handler) CreateProject(ctx context.Context, req gen.CreateProjectRequestObject) (gen.CreateProjectResponseObject, error) {
	b := req.Body
	in := domain.NewProject{Name: b.Name, Identifier: b.Identifier, LeadID: b.ProjectLeadID, Timezone: b.Timezone}
	if b.Description != nil {
		in.Description = *b.Description
	}
	if b.Network != nil {
		n := domain.Network(*b.Network)
		in.Network = &n
	}
	if b.LogoProps != nil {
		in.LogoProps = logoIn(*b.LogoProps)
	}
	p, err := h.uc.CreateProject.Execute(ctx, req.Slug, in)
	if err != nil {
		return nil, err
	}
	return gen.CreateProject201JSONResponse(project(p)), nil
}

// GetProject serves GET /api/v0/projects/{project_id}.
func (h handler) GetProject(ctx context.Context, req gen.GetProjectRequestObject) (gen.GetProjectResponseObject, error) {
	p, err := h.uc.GetProject.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.GetProject200JSONResponse(project(p)), nil
}

// CheckProjectIdentifier serves GET
// /api/v0/workspaces/{slug}/project-identifiers/{identifier}.
func (h handler) CheckProjectIdentifier(ctx context.Context, req gen.CheckProjectIdentifierRequestObject) (gen.CheckProjectIdentifierResponseObject, error) {
	available, err := h.uc.CheckIdentifier.Execute(ctx, req.Slug, req.Identifier)
	if err != nil {
		return nil, err
	}
	return gen.CheckProjectIdentifier200JSONResponse{Available: available}, nil
}

// project is p as the API shows it.
func project(p domain.Project) gen.Project {
	out := gen.Project{
		ID:                   p.ID,
		WorkspaceID:          p.WorkspaceID,
		Name:                 p.Name,
		Description:          p.Description,
		Identifier:           p.Identifier,
		Network:              gen.ProjectNetwork(p.Network),
		ProjectLeadID:        orNull(p.LeadID),
		DefaultAssigneeID:    orNull(p.DefaultAssigneeID),
		CycleView:            p.CycleView,
		ModuleView:           p.ModuleView,
		IssueViewsView:       p.IssueViewsView,
		IntakeView:           p.IntakeView,
		GuestViewAllFeatures: p.GuestViewAllFeatures,
		ArchiveIn:            p.ArchiveIn,
		ArchivedAt:           orNull(p.ArchivedAt),
		LogoProps:            logoOut(p.LogoProps),
		Timezone:             p.Timezone,
		// Required and null until M5.
		CoverImageURL: nullable.NewNullNullable[string](),
		MemberRole:    nullable.NewNullNullable[gen.ProjectRole](),
		SortOrder:     orNull(p.SortOrder),
		// An array, never null: a project without members has [].
		MemberIds: append([]uuid.UUID{}, p.MemberIDs...),
		CreatedAt: p.CreatedAt,
		UpdatedAt: p.UpdatedAt,
	}
	if p.MemberRole != nil {
		out.MemberRole = nullable.NewNullableWithValue(gen.ProjectRole(*p.MemberRole))
	}
	return out
}

// orNull is v as a required field that can be null: null when v is nil.
// The zero Nullable is "unspecified" and would marshal as the zero value.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}

// logoIn is the request's icon as the domain takes it.
func logoIn(l gen.LogoProps) domain.LogoProps {
	var out domain.LogoProps
	if l.InUse != nil {
		inUse := string(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &domain.Emoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &domain.Icon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}

// logoOut is a project's icon as the API shows it: {} for none.
func logoOut(l domain.LogoProps) gen.LogoProps {
	var out gen.LogoProps
	if l.InUse != nil {
		inUse := gen.LogoPropsInUse(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &gen.LogoEmoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &gen.LogoIcon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}
