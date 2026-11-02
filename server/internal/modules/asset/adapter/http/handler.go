package httpadapter

import (
	"context"
	"errors"
	"net/http"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type PrepareUseCase interface {
	Execute(context.Context, domain.NewAsset) (app.PreparedAsset, error)
}
type GetUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.Asset, error)
}
type CompleteUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.Asset, error)
}
type ResolveContentUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.SignedRequest, error)
}
type UseCases struct {
	Prepare        PrepareUseCase
	Get            GetUseCase
	Complete       CompleteUseCase
	ResolveContent ResolveContentUseCase
}
type handler struct{ uc UseCases }

func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: api.Errors.BodyError,
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			api.Errors.Write(w, r, assetBusinessError(err))
		},
	})
	var middlewares []gen.MiddlewareFunc
	for _, middleware := range api.Middlewares(gen.BodyShapes()) {
		middlewares = append(middlewares, middleware)
	}
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{BaseRouter: router, Middlewares: middlewares, ErrorHandlerFunc: api.Errors.BadRequest})
}

type assetProblem struct {
	status       int
	code, detail string
}

func (e assetProblem) Error() string       { return e.detail }
func (e assetProblem) ProblemStatus() int  { return e.status }
func (e assetProblem) ProblemCode() string { return e.code }

// Size and media errors need HTTP categories outside shared's business kinds.
// Every other error retains its owner code or reaches the ordinary 500 handler.
func assetBusinessError(err error) error {
	switch {
	case errors.Is(err, domain.ErrAssetTooLarge):
		return assetProblem{http.StatusRequestEntityTooLarge, domain.CodeAssetTooLarge, "The asset exceeds the allowed file size."}
	case errors.Is(err, domain.ErrAssetTypeUnsupported):
		return assetProblem{http.StatusUnsupportedMediaType, domain.CodeAssetTypeUnsupported, "This purpose requires a JPEG, PNG, WebP or GIF image."}
	default:
		return err
	}
}
