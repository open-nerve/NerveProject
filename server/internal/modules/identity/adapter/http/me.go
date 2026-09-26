package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// GetMe serves GET /api/v0/me.
func (h handler) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	u, err := h.uc.GetMe.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMe200JSONResponse(user(u)), nil
}

// UpdateMe serves PATCH /api/v0/me.
func (h handler) UpdateMe(ctx context.Context, req gen.UpdateMeRequestObject) (gen.UpdateMeResponseObject, error) {
	u, err := h.uc.UpdateMe.Execute(ctx, domain.UserPatch{
		FirstName:   req.Body.FirstName,
		LastName:    req.Body.LastName,
		DisplayName: req.Body.DisplayName,
		Timezone:    req.Body.UserTimezone,
	})
	if err != nil {
		return nil, err
	}
	return gen.UpdateMe200JSONResponse(user(u)), nil
}

// ChangePassword serves POST /api/v0/me/change-password. password_user
// limits it per account (M2 design 3.10): each attempt costs argon2.
func (h handler) ChangePassword(ctx context.Context, req gen.ChangePasswordRequestObject) (gen.ChangePasswordResponseObject, error) {
	if err := h.limitPassword(ctx); err != nil {
		return nil, err
	}
	err := h.uc.ChangePassword.Execute(ctx, app.ChangePasswordInput{Current: req.Body.CurrentPassword, New: req.Body.NewPassword})
	if err != nil {
		return nil, err
	}
	return gen.ChangePassword204Response{}, nil
}

// user is u as the API shows it.
func user(u domain.User) gen.User {
	return gen.User{
		ID:           u.ID,
		Email:        u.Email,
		FirstName:    u.FirstName,
		LastName:     u.LastName,
		DisplayName:  u.DisplayName,
		UserTimezone: u.Timezone,
		// Required and null until M5. The zero Nullable is "unspecified" and
		// would marshal as "": set null explicitly.
		AvatarURL:     nullable.NewNullNullable[string](),
		CoverImageURL: nullable.NewNullNullable[string](),
		CreatedAt:     u.CreatedAt,
	}
}
