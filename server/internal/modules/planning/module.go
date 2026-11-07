// Package planning is the cycles and modules module (M6 design 3, 4): it owns
// the cycles' tables and brings creating, reading, listing, changing,
// archiving, unarchiving and deleting a cycle of a project. Project and
// workspace deletion events enqueue the asynchronous deletion of the deleted
// projects' cycles in the publishing transaction; the 60-day purge removes
// them.
package planning

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts are a project's facts as the module reads them: bootstrap
// converts project's PlanningFacts into them.
type ProjectFacts = app.ProjectFacts

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger *app.Purger
}

// Provide builds planning's purge over pool: its tables, children before
// parents.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{Purger: app.NewPurger([]app.PurgeStep{{Table: "cycles", Delete: store.PurgeCycles}})}
}

// Deps are what the module needs.
type Deps struct {
	Pool *pgxpool.Pool
	// Tx are the transactions of the module's writes, each of which waits
	// for a lock at most issue.write_lock_timeout and answers server_busy
	// past it (bootstrap/gate.go).
	Tx shared.TxManager
	// ReadTx supplies the reads' read-only snapshot transactions, each its
	// own: never one inside a write's.
	ReadTx shared.TxManager
	// WriteGate is the write gate of each project, shared with the
	// work-item side (bootstrap/gate.go).
	WriteGate  app.WriteGate
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Projects are project's PlanningProjects, converted (bootstrap/ports.go).
	Projects app.Projects
	// JobTx runs the deletion job's short transactions.
	JobTx shared.TxManager
	// RuntimeNow is the monotonic clock of the deletion job's budget.
	RuntimeNow   func() time.Time
	DeleteBatch  int
	DeleteBudget time.Duration
	Logger       *slog.Logger
}

// Module is the planning module.
type Module struct {
	uc           httpadapter.UseCases
	deleteRows   *app.DeleteRows
	deleteBudget time.Duration
	logger       *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
	reads := app.NewReads(d.Projects, store, d.Authorizer, d.ReadTx)
	return &Module{
		uc: httpadapter.UseCases{
			ListCycles:     app.NewListCycles(app.ListCyclesDeps{Reads: reads, Cycles: store, Clock: d.Clock}),
			CreateCycle:    app.NewCreateCycle(app.CreateCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			GetCycle:       app.NewGetCycle(app.GetCycleDeps{Reads: reads, Clock: d.Clock}),
			UpdateCycle:    app.NewUpdateCycle(app.UpdateCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			ArchiveCycle:   app.NewArchiveCycle(app.ArchiveCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			UnarchiveCycle: app.NewUnarchiveCycle(app.ArchiveCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
		},
		deleteRows:   app.NewDeleteRows(app.DeleteRowsDeps{Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow, Batch: d.DeleteBatch, Budget: d.DeleteBudget}),
		deleteBudget: d.DeleteBudget, logger: d.Logger,
	}
}

// Register mounts the module's routes on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
