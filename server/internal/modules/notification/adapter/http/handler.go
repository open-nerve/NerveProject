// Package httpadapter serves the notification module's API: it implements
// the strict server that oapi-codegen generates from
// api/modules/notification.yaml into the gen package, and translates
// between the generated types and the use cases.
package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// ListNotificationsUseCase is app.ListNotifications.
type ListNotificationsUseCase interface {
	Execute(ctx context.Context, slug string, q domain.ListQuery, limit *int, cursor *string) (app.NotificationPage, error)
}

// GetUnreadCountUseCase is app.GetUnreadCount.
type GetUnreadCountUseCase interface {
	Execute(ctx context.Context, slug string) (app.UnreadCounts, error)
}

// MarkAllReadUseCase is app.MarkAllRead.
type MarkAllReadUseCase interface {
	Execute(ctx context.Context, slug string, v domain.FilterValues) (int, error)
}

// UpdateNotificationUseCase is app.UpdateNotification.
type UpdateNotificationUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (app.NotificationView, error)
}

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	ListNotifications  ListNotificationsUseCase
	GetUnreadCount     GetUnreadCountUseCase
	MarkAllRead        MarkAllReadUseCase
	UpdateNotification UpdateNotificationUseCase
}

// Register mounts the module's routes on router behind api's per-route
// middlewares; api.Errors answers binding, decoding and handler errors,
// and each workspace operation's query is held to its own parameters
// first (query.go). Every operation needs a token.
func Register(router *httpserver.Router, api *httpserver.API, uc UseCases) {
	strict := gen.NewStrictHandlerWithOptions(handler{uc: uc}, []gen.StrictMiddlewareFunc{withOwnQuery}, gen.StrictHTTPServerOptions{
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
		ErrorHandlerFunc: listBindingError(api),
	})
}

// handler implements gen.StrictServerInterface.
type handler struct {
	uc UseCases
}
