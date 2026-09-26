package httpadapter

import (
	"context"
	"time"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// ListAPITokens serves GET /api/v0/me/api-tokens.
func (h handler) ListAPITokens(ctx context.Context, req gen.ListAPITokensRequestObject) (gen.ListAPITokensResponseObject, error) {
	page, err := h.uc.ListAPITokens.Execute(ctx, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	out := gen.ListAPITokens200JSONResponse{Data: make([]gen.APIToken, len(page.Tokens)), NextCursor: nullable.NewNullNullable[string]()}
	for i, t := range page.Tokens {
		out.Data[i] = gen.APIToken{
			ID: t.ID, Label: t.Label, Description: t.Description,
			ExpiredAt: nullableTime(t.ExpiredAt), LastUsed: nullableTime(t.LastUsed), CreatedAt: t.CreatedAt,
		}
	}
	if page.NextCursor != "" {
		out.NextCursor = nullable.NewNullableWithValue(page.NextCursor)
	}
	return out, nil
}

// CreateAPIToken serves POST /api/v0/me/api-tokens.
func (h handler) CreateAPIToken(ctx context.Context, req gen.CreateAPITokenRequestObject) (gen.CreateAPITokenResponseObject, error) {
	spec := domain.APITokenSpec{Label: req.Body.Label}
	if req.Body.Description != nil {
		spec.Description = *req.Body.Description
	}
	if expires, err := req.Body.ExpiredAt.Get(); err == nil {
		spec.ExpiredAt = &expires
	}
	t, err := h.uc.CreateAPIToken.Execute(ctx, spec)
	if err != nil {
		return nil, err
	}
	return gen.CreateAPIToken201JSONResponse{
		ID: t.ID, Label: t.Label, Description: t.Description,
		ExpiredAt: nullableTime(t.ExpiredAt), LastUsed: nullableTime(t.LastUsed), CreatedAt: t.CreatedAt,
		Token: t.Token,
	}, nil
}

// RevokeAPIToken serves DELETE /api/v0/api-tokens/{token_id}.
func (h handler) RevokeAPIToken(ctx context.Context, req gen.RevokeAPITokenRequestObject) (gen.RevokeAPITokenResponseObject, error) {
	if err := h.uc.RevokeAPIToken.Execute(ctx, req.TokenID); err != nil {
		return nil, err
	}
	return gen.RevokeAPIToken204Response{}, nil
}

// nullableTime is t as a required field that may be null: the zero
// Nullable is "unspecified" and would marshal as the zero time,
// 0001-01-01T00:00:00Z, so nil is set to null explicitly.
func nullableTime(t *time.Time) nullable.Nullable[time.Time] {
	if t == nil {
		return nullable.NewNullNullable[time.Time]()
	}
	return nullable.NewNullableWithValue(*t)
}
