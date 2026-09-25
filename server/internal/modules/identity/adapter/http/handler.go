// Package httpadapter serves the identity module's API: it implements the
// strict server that oapi-codegen generates from api/modules/identity.yaml
// into the gen package, and only translates between the generated types and
// the use cases.
package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// RegisterUseCase is app.Register.
type RegisterUseCase interface {
	Execute(ctx context.Context, in app.RegisterInput) (app.Tokens, error)
}

// GetMeUseCase is app.GetMe.
type GetMeUseCase interface {
	Execute(ctx context.Context) (domain.User, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	Register RegisterUseCase
	GetMe    GetMeUseCase
}

// PublicOperations are the module's routes that need no token (M2 design
// 3.6), as the generated code registers them.
func PublicOperations() []string {
	return []string{"POST /api/v0/auth/register"}
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  api.Errors.BodyError,
		ResponseErrorHandlerFunc: api.Errors.Write,
	})
	var middlewares []gen.MiddlewareFunc
	for _, m := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, m)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       router,
		Middlewares:      middlewares,
		ErrorHandlerFunc: api.Errors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}

// Register serves POST /api/v0/auth/register.
func (h handler) Register(ctx context.Context, req gen.RegisterRequestObject) (gen.RegisterResponseObject, error) {
	meta := httpserver.RequestMetaFrom(ctx)
	tokens, err := h.uc.Register.Execute(ctx, app.RegisterInput{
		Email:     req.Body.Email,
		Password:  req.Body.Password,
		UserAgent: meta.UserAgent,
		IP:        meta.ClientIP,
	})
	if err != nil {
		return nil, err
	}
	return gen.Register201JSONResponse(authTokens(tokens)), nil
}

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

func authTokens(t app.Tokens) gen.AuthTokens {
	return gen.AuthTokens{
		TokenType:             gen.AuthTokensTokenTypeBearer,
		AccessToken:           t.AccessToken,
		AccessTokenExpiresIn:  int(t.AccessExpiresIn.Seconds()),
		RefreshToken:          t.RefreshToken,
		RefreshTokenExpiresAt: t.RefreshExpiresAt,
	}
}
