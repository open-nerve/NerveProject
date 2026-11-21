// Package httpadapter serves the navigation module's API: it implements the
// strict server that oapi-codegen generates from api/modules/navigation.yaml
// into the gen package, and translates between the generated types and the
// use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// ListProjectViewsUseCase is app.ListProjectViews.
type ListProjectViewsUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID) ([]domain.View, error)
}

// ListWorkspaceViewsUseCase is app.ListWorkspaceViews.
type ListWorkspaceViewsUseCase interface {
	Execute(ctx context.Context, slug string) ([]domain.View, error)
}

// GetViewUseCase is app.GetView.
type GetViewUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) (domain.View, error)
}

// CreateProjectViewUseCase is app.CreateProjectView.
type CreateProjectViewUseCase interface {
	Execute(ctx context.Context, projectID uuid.UUID, n domain.NewView) (domain.View, error)
}

// CreateWorkspaceViewUseCase is app.CreateWorkspaceView.
type CreateWorkspaceViewUseCase interface {
	Execute(ctx context.Context, slug string, n domain.NewView) (domain.View, error)
}

// UpdateViewUseCase is app.UpdateView.
type UpdateViewUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (domain.View, error)
}

// DeleteViewUseCase is app.DeleteView.
type DeleteViewUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// UseCases are the use cases behind the module's operations: the views'
// and the favorites' (favorites.go).
type UseCases struct {
	ListProjectViews    ListProjectViewsUseCase
	ListWorkspaceViews  ListWorkspaceViewsUseCase
	GetView             GetViewUseCase
	CreateProjectView   CreateProjectViewUseCase
	CreateWorkspaceView CreateWorkspaceViewUseCase
	UpdateView          UpdateViewUseCase
	DeleteView          DeleteViewUseCase
	ListFavorites       ListFavoritesUseCase
	CreateFavorite      CreateFavoriteUseCase
	UpdateFavorite      UpdateFavoriteUseCase
	DeleteFavorite      DeleteFavoriteUseCase
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
