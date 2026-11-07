// Package httpadapter serves the planning module's API: it implements the
// strict server that oapi-codegen generates from api/modules/planning.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// CreateCycleUseCase is app.CreateCycle.
type CreateCycleUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, in domain.NewCycle) (domain.ReadCycle, error)
}

// ListCyclesUseCase is app.ListCycles.
type ListCyclesUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, archived bool) ([]domain.ReadCycle, error)
}

// GetCycleUseCase is app.GetCycle.
type GetCycleUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error)
}

// UpdateCycleUseCase is app.UpdateCycle.
type UpdateCycleUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.CyclePatch) (domain.ReadCycle, error)
}

// ArchiveCycleUseCase is app.ArchiveCycle, which archives or unarchives.
type ArchiveCycleUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.ReadCycle, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	ListCycles     ListCyclesUseCase
	CreateCycle    CreateCycleUseCase
	GetCycle       GetCycleUseCase
	UpdateCycle    UpdateCycleUseCase
	ArchiveCycle   ArchiveCycleUseCase
	UnarchiveCycle ArchiveCycleUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors.
// Every operation needs a token.
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
