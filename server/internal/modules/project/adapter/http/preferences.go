package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// GetProjectPreferences serves GET /api/v0/me/projects/{project_id}/preferences.
func (h handler) GetProjectPreferences(ctx context.Context, req gen.GetProjectPreferencesRequestObject) (gen.GetProjectPreferencesResponseObject, error) {
	p, err := h.uc.GetPreferences.Execute(ctx, req.ProjectID)
	if err != nil {
		return nil, err
	}
	return gen.GetProjectPreferences200JSONResponse(preferencesOut(p)), nil
}

// UpdateProjectPreferences serves PATCH
// /api/v0/me/projects/{project_id}/preferences: the fields the body names go
// to the use case, the navigation whole; a tab the contract does not list
// goes too, and the domain refuses it (422).
func (h handler) UpdateProjectPreferences(ctx context.Context, req gen.UpdateProjectPreferencesRequestObject) (gen.UpdateProjectPreferencesResponseObject, error) {
	b := req.Body
	in := domain.PreferencesPatch{SortOrder: b.SortOrder}
	if n := b.Navigation; n != nil {
		in.Navigation = &domain.Navigation{DefaultTab: string(n.DefaultTab), HideInMoreMenu: make([]string, len(n.HideInMoreMenu))}
		for i, tab := range n.HideInMoreMenu {
			in.Navigation.HideInMoreMenu[i] = string(tab)
		}
	}
	p, err := h.uc.UpdatePreferences.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateProjectPreferences200JSONResponse(preferencesOut(p)), nil
}

// preferencesOut is p as the API shows it: the hidden tabs an array, never
// null.
func preferencesOut(p domain.Preferences) gen.ProjectPreferences {
	out := gen.ProjectPreferences{SortOrder: p.SortOrder, Navigation: gen.ProjectNavigation{
		DefaultTab: gen.ProjectTab(p.Navigation.DefaultTab), HideInMoreMenu: make([]gen.ProjectTab, len(p.Navigation.HideInMoreMenu)),
	}}
	for i, tab := range p.Navigation.HideInMoreMenu {
		out.Navigation.HideInMoreMenu[i] = gen.ProjectTab(tab)
	}
	return out
}
