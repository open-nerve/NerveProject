// Package httpadapter serves the instance module's API: it implements the
// strict server that oapi-codegen generates from api/modules/instance.yaml
// into the gen package.
package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// UseCases are the use cases behind the module's operations.
type UseCases struct {
	GetInfo       *app.GetInfo
	ListTimezones *app.ListTimezones
}

// PublicOperations are the module's routes that need no token (M2 design
// 3.6), as the generated code registers them.
func PublicOperations() []string {
	return []string{"GET /api/v0/instance", "GET /api/v0/timezones"}
}

// Register mounts the module's routes on router, the root router from
// httpserver.NewRouter, behind the platform's per-route middlewares. They
// are more specific than the platform's /api/ fallback, which keeps
// answering every other API path with a 404 problem. Binding, decoding and
// handler errors are answered as problem+json by api.Errors.
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

// handler implements gen.StrictServerInterface: it only translates between
// the generated types and the use cases.
type handler struct {
	uc UseCases
}

// GetInstance serves GET /api/v0/instance.
func (h handler) GetInstance(context.Context, gen.GetInstanceRequestObject) (gen.GetInstanceResponseObject, error) {
	info := h.uc.GetInfo.Execute()
	return gen.GetInstance200JSONResponse{
		Product:    info.Product,
		Version:    info.Version,
		Commit:     info.Commit,
		APIVersion: gen.InstanceInfoAPIVersion(info.APIVersion),

		SignupEnabled:            info.SignupEnabled,
		WorkspaceCreationEnabled: info.WorkspaceCreationEnabled,
		FileSizeLimit:            int(info.FileSizeLimit),
	}, nil
}

// ListTimezones serves GET /api/v0/timezones.
func (h handler) ListTimezones(context.Context, gen.ListTimezonesRequestObject) (gen.ListTimezonesResponseObject, error) {
	zones, err := h.uc.ListTimezones.Execute()
	if err != nil {
		return nil, err
	}
	out := gen.ListTimezones200JSONResponse{Data: make([]gen.Timezone, len(zones))}
	for i, z := range zones {
		out.Data[i] = gen.Timezone{Label: z.Label, Value: z.Name, UtcOffset: "UTC" + z.Offset, GmtOffset: "GMT" + z.Offset}
	}
	return out, nil
}
