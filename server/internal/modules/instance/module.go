// Package instance is the pilot module and the template for every module
// (M0 design 3.2): GET /api/v0/instance tells API clients what this nerve
// instance runs and how it is set up, and GET /api/v0/timezones lists the
// time zones to choose from (M2 design 5.3).
package instance

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/buildinfo"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/instance/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Deps are what bootstrap gives the module: the settings the instance
// reports, from the configuration, and the clock.
type Deps struct {
	SignupEnabled            bool  // auth.signup_enabled
	WorkspaceCreationEnabled bool  // workspace.creation_enabled
	FileSizeLimit            int64 // files.size_limit, in bytes
	Clock                    app.Clock
}

// Module is the wired instance module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module: GetInfo reads the build of the running binary.
func New(d Deps) *Module {
	settings := domain.Settings{
		SignupEnabled:            d.SignupEnabled,
		WorkspaceCreationEnabled: d.WorkspaceCreationEnabled,
		FileSizeLimit:            d.FileSizeLimit,
	}
	return &Module{uc: httpadapter.UseCases{
		GetInfo:       app.NewGetInfo(buildinfo.Source{}, settings),
		ListTimezones: app.NewListTimezones(d.Clock),
	}}
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
