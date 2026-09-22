// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs.
package instance

import (
	"net/http"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Module is the wired instance module.
type Module struct {
	getInfo *app.GetInfo
}

// New wires the module: GetInfo reads the build of the running binary.
func New() *Module {
	return &Module{getInfo: app.NewGetInfo(buildinfo.Source{})}
}

// Register mounts the module's API on mux, the root router from
// httpserver.NewMux; apiErrors answers binding and handler errors.
func (m *Module) Register(mux *http.ServeMux, apiErrors httpserver.APIErrors) {
	httpadapter.Register(mux, m.getInfo, apiErrors)
}
