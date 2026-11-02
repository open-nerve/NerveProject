package asset

import (
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

// Deps exposes the ports consumed by the local transfer handler. Bootstrap owns
// the underlying storage resource and closes it after HTTP and jobs stop.
type Deps = httpadapter.Deps

type Module struct {
	transfer *httpadapter.Transfer
}

func New(deps Deps) (*Module, error) {
	transfer, err := httpadapter.NewTransfer(deps)
	if err != nil {
		return nil, err
	}
	return &Module{transfer: transfer}, nil
}

func (m *Module) Register(router *httpserver.Router, api *httpserver.API, limiter httpserver.Limiter) {
	m.transfer.Register(router, api.TransferMiddleware(limiter))
}
