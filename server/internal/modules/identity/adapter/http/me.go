package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
)

// GetMe serves GET /api/v0/me.
func (h handler) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	u, err := h.uc.GetMe.Execute(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMe200JSONResponse{
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
	}, nil
}
