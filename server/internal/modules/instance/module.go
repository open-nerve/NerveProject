// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs.
package instance

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Module is the wired instance module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module: GetInfo reads the build of the running binary.
func New() *Module {
	return &Module{uc: httpadapter.UseCases{GetInfo: app.NewGetInfo(buildinfo.Source{})}}
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Register mounts the module's API on router, the root router from
// httpserver.NewRouter, behind api's per-route middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}
