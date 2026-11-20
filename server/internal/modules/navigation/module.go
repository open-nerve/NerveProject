// Package navigation is the saved views module (M7 design 3.7, 4): it owns
// the issue_views table, each row a saved view, a project's or, without a
// project, a workspace's: its name, description, icon and the list
// settings it shows its work items with, and its owner. Its use cases list
// a project's and a workspace's views, read one by its id and create them
// (P3b spec 2.4, 2.5). A project's or a workspace's deletion enqueues, in
// its transaction, the deletion of each deleted project's views and of the
// workspace's own (P3b spec 2.7); the 60-day purge removes the deleted ones
// (Provide).
package navigation

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger *app.Purger
}

// Provide builds, over pool, the navigation module's purge: its one table,
// which no other table references, so the module purges before the issue
// module (M7 design 4.3, P3b spec 2.7).
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{{Table: "issue_views", Delete: store.PurgeIssueViews}}),
	}
}

// ProjectFacts are the facts of a project the module reads (Projects):
// bootstrap converts project's NavigationFacts into them.
type ProjectFacts = app.ProjectFacts

// Deps are what the module needs.
type Deps struct {
	Pool *pgxpool.Pool
	// Tx are the transactions of the views' writes, which enter no write
	// gate and answer no server_busy (P3b spec 2.4).
	Tx shared.TxManager
	// ReadTx supplies the reads' read-only snapshot transactions, each its
	// own.
	ReadTx     shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Projects are project's NavigationProjects, converted
	// (bootstrap/navigation_ports.go).
	Projects app.Projects
	// Workspaces are workspace's WorkspaceDirectory, converted
	// (bootstrap/navigation_ports.go).
	Workspaces app.Workspaces
	// ListSettings and WorkspaceListSettings check a project's and a
	// workspace's views' words: the issue module's two, as issue.Provide
	// offers them (P3b spec 2.4).
	ListSettings, WorkspaceListSettings app.ListSettings
	// JobTx runs the deletion job's short transactions: each lock wait at
	// most issue.write_lock_timeout, no write gate, and a lock failure kept
	// as PostgreSQL's for River's retry (bootstrap/gate.go).
	JobTx shared.TxManager
	// RuntimeNow is the monotonic clock of the deletion job's budget.
	RuntimeNow   func() time.Time
	DeleteBatch  int
	DeleteBudget time.Duration
	Logger       *slog.Logger
}

// Module is the navigation module.
type Module struct {
	listProjectViews    *app.ListProjectViews
	listWorkspaceViews  *app.ListWorkspaceViews
	getView             *app.GetView
	createProjectView   *app.CreateProjectView
	createWorkspaceView *app.CreateWorkspaceView
	deleteRows          *app.DeleteRows
	deleteBudget        time.Duration
	logger              *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	reads := app.NewReads(d.Workspaces, d.Projects, store, d.Authorizer, d.ReadTx)
	creations := app.CreateViewDeps{Locks: app.NewLocks(d.Workspaces, d.Projects, d.Authorizer, d.Tx), Views: store, Clock: d.Clock,
		Words: app.Words{Project: d.ListSettings, Workspace: d.WorkspaceListSettings}}
	return &Module{
		listProjectViews: app.NewListProjectViews(reads), listWorkspaceViews: app.NewListWorkspaceViews(reads), getView: app.NewGetView(reads),
		createProjectView: app.NewCreateProjectView(creations), createWorkspaceView: app.NewCreateWorkspaceView(creations),
		deleteRows: app.NewDeleteRows(app.DeleteRowsDeps{Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow, Batch: d.DeleteBatch,
			Budget: d.DeleteBudget}),
		deleteBudget: d.DeleteBudget, logger: d.Logger,
	}
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
