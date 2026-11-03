package httpadapter

import (
	"context"
	"time"
	"uuid"

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
	avatarSet, avatar := imageIDPatch(req.Body.AvatarAssetID)
	coverSet, cover := imageIDPatch(req.Body.CoverImageAssetID)
	u, err := h.uc.UpdateMe.Execute(ctx, domain.UserPatch{
		SetAvatar: avatarSet, AvatarAssetID: avatar, SetCoverImage: coverSet, CoverImageAssetID: cover,
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

// DeactivateMe serves POST /api/v0/me/deactivate.
func (h handler) DeactivateMe(ctx context.Context, _ gen.DeactivateMeRequestObject) (gen.DeactivateMeResponseObject, error) {
	if err := h.uc.Deactivate.Execute(ctx); err != nil {
		return nil, err
	}
	return gen.DeactivateMe204Response{}, nil
}

// user is u as the API shows it.
func user(u domain.User) gen.User {
	avatar, avatarExpires := imageReply(u.Avatar)
	cover, coverExpires := imageReply(u.CoverImage)
	return gen.User{
		ID:                u.ID,
		Email:             u.Email,
		FirstName:         u.FirstName,
		LastName:          u.LastName,
		DisplayName:       u.DisplayName,
		UserTimezone:      u.Timezone,
		AvatarAssetID:     nullableImageID(u.AvatarAssetID),
		CoverImageAssetID: nullableImageID(u.CoverImageAssetID),
		AvatarURL:         avatar, AvatarURLExpiresAt: avatarExpires,
		CoverImageURL: cover, CoverImageURLExpiresAt: coverExpires,
		CreatedAt: u.CreatedAt,
	}
}

func imageIDPatch(v nullable.Nullable[uuid.UUID]) (bool, *uuid.UUID) {
	if !v.IsSpecified() {
		return false, nil
	}
	if id, err := v.Get(); err == nil {
		return true, &id
	}
	return true, nil
}
func nullableImageID(id *uuid.UUID) nullable.Nullable[uuid.UUID] {
	if id == nil {
		return nullable.NewNullNullable[uuid.UUID]()
	}
	return nullable.NewNullableWithValue(*id)
}
func imageReply(image *domain.ImageURL) (nullable.Nullable[string], nullable.Nullable[time.Time]) {
	if image == nil {
		return nullable.NewNullNullable[string](), nullable.NewNullNullable[time.Time]()
	}
	return nullable.NewNullableWithValue(image.URL), nullable.NewNullableWithValue(image.ExpiresAt)
}
