package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// GetWorkspacePreferences serves GET /api/v0/me/workspaces/{slug}/preferences.
func (h handler) GetWorkspacePreferences(ctx context.Context, req gen.GetWorkspacePreferencesRequestObject) (gen.GetWorkspacePreferencesResponseObject, error) {
	p, err := h.uc.GetPreferences.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	return gen.GetWorkspacePreferences200JSONResponse(preferences(p)), nil
}

// UpdateWorkspacePreferences serves PATCH /api/v0/me/workspaces/{slug}/preferences.
func (h handler) UpdateWorkspacePreferences(ctx context.Context, req gen.UpdateWorkspacePreferencesRequestObject) (gen.UpdateWorkspacePreferencesResponseObject, error) {
	patch := domain.PreferencesPatch{NavigationProjectLimit: req.Body.NavigationProjectLimit}
	if mode := req.Body.NavigationControlPreference; mode != nil {
		m := string(*mode)
		patch.NavigationControl = &m
	}
	p, err := h.uc.UpdatePreferences.Execute(ctx, req.Slug, patch)
	if err != nil {
		return nil, err
	}
	return gen.UpdateWorkspacePreferences200JSONResponse(preferences(p)), nil
}

// preferences is p as the API shows it.
func preferences(p domain.Preferences) gen.WorkspacePreferences {
	return gen.WorkspacePreferences{
		NavigationControlPreference: gen.NavigationControlPreference(p.NavigationControl),
		NavigationProjectLimit:      p.NavigationProjectLimit,
	}
}
