package httpadapter

import (
	"context"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetProfile serves GET /api/v0/me/profile.
func (h handler) GetProfile(ctx context.Context, _ gen.GetProfileRequestObject) (gen.GetProfileResponseObject, error) {
	p, err := h.uc.GetProfile.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetProfile200JSONResponse(profile(p)), nil
}

// UpdateProfile serves PATCH /api/v0/me/profile.
func (h handler) UpdateProfile(ctx context.Context, req gen.UpdateProfileRequestObject) (gen.UpdateProfileResponseObject, error) {
	b := req.Body
	patch := domain.ProfilePatch{
		Theme:           (*string)(b.Theme),
		Language:        (*string)(b.Language),
		StartOfTheWeek:  (*int)(b.StartOfTheWeek),
		IsOnboarded:     b.IsOnboarded,
		IsTourCompleted: b.IsTourCompleted,
	}
	if b.OnboardingStep != nil {
		patch.OnboardingStep = domain.OnboardingStepsPatch(*b.OnboardingStep)
	}
	// Absent keeps the workspace, null clears it.
	if b.LastWorkspaceID.IsSpecified() {
		patch.LastWorkspaceSet = true
		if id, err := b.LastWorkspaceID.Get(); err == nil {
			patch.LastWorkspaceID = &id
		}
	}
	p, err := h.uc.UpdateProfile.Execute(ctx, patch)
	if err != nil {
		return nil, err
	}
	return gen.UpdateProfile200JSONResponse(profile(p)), nil
}

// profile is p as the API shows it.
func profile(p domain.Profile) gen.Profile {
	workspace := nullable.NewNullNullable[uuid.UUID]()
	if p.LastWorkspaceID != nil {
		workspace = nullable.NewNullableWithValue(*p.LastWorkspaceID)
	}
	return gen.Profile{
		Theme:           gen.Theme(p.Theme),
		Language:        gen.Language(p.Language),
		StartOfTheWeek:  gen.StartOfTheWeek(p.StartOfTheWeek),
		OnboardingStep:  gen.OnboardingSteps(p.OnboardingStep),
		IsOnboarded:     p.IsOnboarded,
		IsTourCompleted: p.IsTourCompleted,
		LastWorkspaceID: workspace,
		UpdatedAt:       p.UpdatedAt,
	}
}
