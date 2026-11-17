// Package notification is the notifications module (M7 design 3.2–3.4, 4):
// it owns the notifications table, each row one account's notification of
// one change of a work item, with the snapshot of the work item and the
// change at the moment it was sent. A work item's deletion deletes its
// notifications in its transaction; a project's or a workspace's deletion
// enqueues the deletion of each deleted project's notifications, which waits
// for the project's work items to be deleted (M7 P1b spec 2.8). The 60-day
// purge removes the deleted ones (Provide).
package notification

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger *app.Purger
	store  *postgresadapter.Store
}

// Provide builds, over pool, the notification module's purge: its one
// table, which no other table references, so the module purges before the
// issue module (M7 design 4.3); and the store its subscribers write through
// in the publishing transaction.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{{Table: "notifications", Delete: store.PurgeNotifications}}),
		store:  store,
	}
}

// Deps are what the module needs.
type Deps struct {
	Pool *pgxpool.Pool
	// Issues is the issue module's NotificationIssues, which bootstrap hands
	// over: whether a project has an undeleted work item.
	Issues app.ProjectIssues
	// JobTx runs the deletion job's short transactions: each lock wait at
	// most issue.write_lock_timeout, no write gate, and a lock failure kept
	// as PostgreSQL's for River's retry (bootstrap/gate.go).
	JobTx shared.TxManager
	// RuntimeNow is the monotonic clock of the deletion job's budget and of
	// its wait for the project's work items.
	RuntimeNow   func() time.Time
	DeleteBatch  int
	DeleteBudget time.Duration
	Logger       *slog.Logger
}

// Module is the notification module.
type Module struct {
	deleteRows   *app.DeleteRows
	deleteBudget time.Duration
	logger       *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	return &Module{
		deleteRows: app.NewDeleteRows(app.DeleteRowsDeps{Issues: d.Issues, Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow,
			Batch: d.DeleteBatch, Budget: d.DeleteBudget}),
		deleteBudget: d.DeleteBudget, logger: d.Logger,
	}
}
