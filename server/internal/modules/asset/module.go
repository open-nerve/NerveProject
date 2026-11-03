package asset

import (
	"errors"
	"time"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Bootstrap opens storage first, then constructs the module once all owners
// are available. Storage remains owned by bootstrap until HTTP and jobs stop.
type Deps struct {
	Events       shared.EventPublisher
	Files        config.FilesConfig
	RuntimeNow   func() time.Time
	Transfer     *httpadapter.Deps
	Ports        Provided
	Access       app.AccessDeps
	Parents      app.ParentDeletionReader
	Objects      app.ObjectDeleter
	Inventory    app.ObjectLister
	Temporary    app.TemporaryCleaner
	JobTx        shared.TxManager
	ReadTx       shared.TxManager
	UploadSigner app.UploadSigner
	ReadSigner   app.ReadSigner
	Sealing      app.SealingDeps
	PurgeAfter   time.Duration
}
type Module struct {
	transfer                       *httpadapter.Transfer
	useCases                       httpadapter.UseCases
	cleanup                        *app.Cleanup
	deleteScope                    *app.DeleteScope
	cleanupInterval, cleanupBudget time.Duration
}

func New(d Deps) (*Module, error) {
	if d.PurgeAfter <= 0 {
		return nil, errors.New("asset retention must be positive")
	}
	if d.Events == nil || d.Ports.store == nil || d.Parents == nil || d.Objects == nil || d.Inventory == nil || d.Temporary == nil || d.JobTx == nil || d.ReadTx == nil || d.UploadSigner == nil || d.ReadSigner == nil || d.Sealing.Inspector == nil || d.Sealing.Reader == nil || d.Sealing.Sealer == nil || d.Sealing.Validator == nil {
		return nil, errors.New("asset module lacks lifecycle dependencies")
	}
	if d.RuntimeNow == nil {
		return nil, errors.New("asset module lacks runtime clock")
	}
	if _, err := d.Files.TransferBudget(d.Files.SizeLimit); err != nil {
		return nil, err
	}
	var transfer *httpadapter.Transfer
	if d.Transfer != nil {
		var err error
		transfer, err = httpadapter.NewTransfer(*d.Transfer)
		if err != nil {
			return nil, err
		}
	}
	d.Access.Assets = d.Ports.store
	access := app.NewAccess(d.Access)
	files := d.Files
	cleanup := app.NewCleanup(app.CleanupDeps{Rows: d.Ports.store, Locks: d.Ports.store, Writer: d.Ports.store, Parents: d.Parents, Objects: d.Objects, Inventory: d.Inventory, Temporary: d.Temporary, Tx: d.JobTx, Clock: d.Access.Clock, RuntimeNow: d.RuntimeNow, Batch: files.CleanupBatch, Budget: files.CleanupBudget, OperationTimeout: files.OperationTimeout, PurgeAfter: d.PurgeAfter})
	deletion := app.NewDeleteScope(app.DeleteScopeDeps{Rows: d.Ports.store, Locks: d.Ports.store, Writer: d.Ports.store, Tx: d.JobTx, Batch: files.CleanupBatch, Budget: files.CleanupBudget})
	publication := app.AttachmentPublication{Writer: d.Ports.store, Events: d.Events}
	uc := httpadapter.UseCases{
		Duplicate:      app.NewDuplicate(app.DuplicateDeps{Publication: publication, Rows: d.Ports.store, Creator: d.Ports.store, Writer: d.Ports.store, Access: access, Storage: d.Sealing, SizeLimit: files.SizeLimit, OperationTimeout: files.OperationTimeout, RuntimeNow: d.RuntimeNow}),
		Delete:         app.NewDelete(app.DeleteDeps{Events: d.Events, Rows: d.Ports.store, Writer: d.Ports.store, Access: access}),
		Complete:       app.NewComplete(app.CompleteDeps{Publication: publication, Rows: d.Ports.store, Writer: d.Ports.store, Access: access, Storage: d.Sealing, SizeLimit: files.SizeLimit, OperationTimeout: files.OperationTimeout, RuntimeNow: d.RuntimeNow}),
		Prepare:        app.NewPrepare(app.PrepareDeps{Access: access, Rows: d.Ports.store, Signer: d.UploadSigner, Budget: files, SizeLimit: files.SizeLimit, UploadTTL: files.UploadTTL, RuntimeNow: d.RuntimeNow}),
		Get:            app.NewGet(app.GetDeps{Rows: d.Ports.store, Hosts: d.Access.Hosts, ReadTx: d.ReadTx}),
		ResolveContent: app.NewResolveContent(app.ResolveContentDeps{Rows: d.Ports.store, Access: access, Signer: d.ReadSigner, RuntimeNow: d.RuntimeNow}),
	}
	return &Module{transfer: transfer, useCases: uc, cleanup: cleanup, deleteScope: deletion, cleanupInterval: files.CleanupInterval, cleanupBudget: files.CleanupBudget}, nil
}
func (m *Module) Register(router *httpserver.Router, api *httpserver.API, limiter httpserver.Limiter) {
	httpadapter.Register(router, api, m.useCases)
	if m.transfer != nil {
		m.transfer.Register(router, api.TransferMiddleware(limiter))
	} else {
		httpadapter.RegisterStorageNamespace(router, api.TransferMiddleware(limiter))
	}
}
