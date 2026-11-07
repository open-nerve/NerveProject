package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
)

// ListModuleLinks serves GET /api/v0/modules/{module_id}/links: data an
// array, never null.
func (h handler) ListModuleLinks(ctx context.Context, req gen.ListModuleLinksRequestObject) (gen.ListModuleLinksResponseObject, error) {
	links, err := h.uc.ListModuleLinks.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	out := gen.ModuleLinkCollection{Data: make([]gen.ModuleLink, len(links))}
	for i, l := range links {
		out.Data[i] = moduleLink(l)
	}
	return gen.ListModuleLinks200JSONResponse(out), nil
}

// CreateModuleLink serves POST /api/v0/modules/{module_id}/links: the
// address and the title go to the use case as given, a title given as null
// as none.
func (h handler) CreateModuleLink(ctx context.Context, req gen.CreateModuleLinkRequestObject) (gen.CreateModuleLinkResponseObject, error) {
	_, title := given(req.Body.Title)
	l, err := h.uc.CreateModuleLink.Execute(ctx, req.ModuleID, domain.NewModuleLink{URL: req.Body.URL, Title: title})
	if err != nil {
		return nil, err
	}
	return gen.CreateModuleLink201JSONResponse(moduleLink(l)), nil
}

// UpdateModuleLink serves PATCH /api/v0/module-links/{module_link_id}: what
// is given goes to the use case, a title given as null to clear it.
func (h handler) UpdateModuleLink(ctx context.Context, req gen.UpdateModuleLinkRequestObject) (gen.UpdateModuleLinkResponseObject, error) {
	p := domain.ModuleLinkPatch{URL: req.Body.URL}
	p.SetTitle, p.Title = given(req.Body.Title)
	l, err := h.uc.UpdateModuleLink.Execute(ctx, req.ModuleLinkID, p)
	if err != nil {
		return nil, err
	}
	return gen.UpdateModuleLink200JSONResponse(moduleLink(l)), nil
}

// DeleteModuleLink serves DELETE /api/v0/module-links/{module_link_id}.
func (h handler) DeleteModuleLink(ctx context.Context, req gen.DeleteModuleLinkRequestObject) (gen.DeleteModuleLinkResponseObject, error) {
	if err := h.uc.DeleteModuleLink.Execute(ctx, req.ModuleLinkID); err != nil {
		return nil, err
	}
	return gen.DeleteModuleLink204Response{}, nil
}

// moduleLink is l as the API shows it: null for each value it has not, its
// metadata with the keys it has alone.
func moduleLink(l domain.ModuleLink) gen.ModuleLink {
	m := apigen.LinkMetadata{}
	if value, ok := l.Metadata["title"]; ok {
		m.Title = orNull(value)
	}
	if value, ok := l.Metadata["favicon"]; ok {
		m.Favicon = orNull(value)
	}
	if value, ok := l.Metadata["url"]; ok {
		m.URL = orNull(value)
	}
	if value, ok := l.Metadata["favicon_url"]; ok {
		m.FaviconURL = orNull(value)
	}
	if value, ok := l.Metadata["error"]; ok {
		m.Error = orNull((*apigen.LinkMetadataError)(value))
	}
	return gen.ModuleLink{ID: l.ID, ModuleID: l.ModuleID, URL: l.URL, Title: orNull(l.Title), Metadata: m, CreatedByID: orNull(l.CreatedByID),
		CreatedAt: l.CreatedAt, UpdatedAt: l.UpdatedAt}
}
