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

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

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
	deleteRows   *app.DeleteRows
	deleteBudget time.Duration
	logger       *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{
		deleteRows:   app.NewDeleteRows(app.DeleteRowsDeps{Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow, Batch: d.DeleteBatch, Budget: d.DeleteBudget}),
		deleteBudget: d.DeleteBudget, logger: d.Logger,
	}
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
