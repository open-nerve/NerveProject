// Package httpadapter serves the instance module's API: it implements the
// strict server that oapi-codegen generates from api/modules/instance.yaml
// into the gen package.
package httpadapter

import (
	"context"
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Register mounts the module's routes on mux, the root router from
// httpserver.NewMux. They are more specific than the platform's /api/
// fallback, which keeps answering every other API path with a 404 problem.
// Binding and handler errors are answered as problem+json by apiErrors.
func Register(mux *http.ServeMux, getInfo *app.GetInfo, apiErrors httpserver.APIErrors) {
	strict := gen.NewStrictHandlerWithOptions(handler{getInfo: getInfo}, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  apiErrors.BadRequest,
		ResponseErrorHandlerFunc: apiErrors.InternalError,
	})
	gen.HandlerWithOptions(strict, gen.StdHTTPServerOptions{
		BaseRouter:       mux,
		ErrorHandlerFunc: apiErrors.BadRequest,
	})
}

// handler implements gen.StrictServerInterface: it only translates between
// the generated types and the use cases.
type handler struct {
	getInfo *app.GetInfo
}

// GetInstance serves GET /api/v0/instance.
func (h handler) GetInstance(context.Context, gen.GetInstanceRequestObject) (gen.GetInstanceResponseObject, error) {
	info := h.getInfo.Execute()
	return gen.GetInstance200JSONResponse{
		Product:    info.Product,
		Version:    info.Version,
		Commit:     info.Commit,
		APIVersion: gen.InstanceInfoAPIVersion(info.APIVersion),
	}, nil
}
