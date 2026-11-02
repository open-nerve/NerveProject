package asset

import (
	"errors"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Bootstrap opens storage first, then constructs the module once all owners
// are available. Storage remains owned by bootstrap until HTTP and jobs stop.
type Deps struct {
	Transfer httpadapter.Deps
	Ports    Provided
	Access   app.AccessDeps
	Parents  app.ParentDeletionReader
	Objects  app.ObjectDeleter
	JobTx    shared.TxManager
}
type Module struct {
	transfer                       *httpadapter.Transfer
	access                         *app.Access
	cleanup                        *app.Cleanup
	deleteScope                    *app.DeleteScope
	cleanupInterval, cleanupBudget time.Duration
}

func New(d Deps) (*Module, error) {
	if d.Ports.store == nil || d.Parents == nil || d.Objects == nil || d.JobTx == nil {
		return nil, errors.New("asset module lacks lifecycle dependencies")
	}
	transfer, err := httpadapter.NewTransfer(d.Transfer)
	if err != nil {
		return nil, err
	}
	d.Access.Assets = d.Ports.store
	files := d.Transfer.Files
	cleanup := app.NewCleanup(app.CleanupDeps{Rows: d.Ports.store, Locks: d.Ports.store, Writer: d.Ports.store, Parents: d.Parents, Objects: d.Objects, Tx: d.JobTx, Clock: d.Access.Clock, RuntimeNow: d.Transfer.RuntimeNow, Batch: files.CleanupBatch, Budget: files.CleanupBudget, OperationTimeout: files.OperationTimeout})
	deletion := app.NewDeleteScope(app.DeleteScopeDeps{Rows: d.Ports.store, Locks: d.Ports.store, Writer: d.Ports.store, Tx: d.JobTx, Batch: files.CleanupBatch, Budget: files.CleanupBudget})
	return &Module{transfer: transfer, access: app.NewAccess(d.Access), cleanup: cleanup, deleteScope: deletion, cleanupInterval: files.CleanupInterval, cleanupBudget: files.CleanupBudget}, nil
}
func (m *Module) Register(router *httpserver.Router, api *httpserver.API, limiter httpserver.Limiter) {
	m.transfer.Register(router, api.TransferMiddleware(limiter))
}
