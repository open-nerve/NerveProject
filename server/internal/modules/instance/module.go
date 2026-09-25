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
	getInfo *app.GetInfo
}

// New wires the module: GetInfo reads the build of the running binary.
func New() *Module {
	return &Module{getInfo: app.NewGetInfo(buildinfo.Source{})}
}

// Register mounts the module's API on router, the root router from
// httpserver.NewRouter; apiErrors answers binding and handler errors.
func (m *Module) Register(router *httpserver.Router, apiErrors httpserver.APIErrors) {
	httpadapter.Register(router, m.getInfo, apiErrors)
}
