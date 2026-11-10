// Package planning is the cycles and modules module (M6 design 3, 4): it owns
// the cycles' and the modules' tables and brings creating, reading, listing,
// changing, archiving, unarchiving and deleting a cycle of a project;
// creating, listing, reading, changing, archiving, unarchiving and deleting a
// module of a project, and adding, listing, changing and removing a module's
// links; reading a cycle's and a module's progress, and the numbers of their
// list rows, computed from the issue module's facts of their work items
// (IssueStats); and reading and changing each caller's own display settings on
// a cycle's and a module's page. A cycle's and a module's deletion publish it
// in their transaction (EventTypes), which the issue module's subscriber takes
// the work items out of it in. A link's creation and a change of its address
// enqueue its crawl in their transaction, whose job writes the link's
// metadata. A periodic job freezes, once, the progress of each cycle that
// has ended, which a cycle's progress and list row answer beside the live
// one (FreezeCycles). Project and workspace deletion events enqueue the
// asynchronous deletion of the deleted projects' planning rows in the
// publishing transaction; the 60-day purge removes them. It tells the issue
// module about the cycles and modules a work item's write names and those
// that keep work items from automatic archival (IssueTargets), and the names
// of the cycles and modules of the projects a list of work items ordered by
// them reads (IssueDirectory).
package planning

import (
	"context"
	"log/slog"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres"
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ProjectFacts are a project's facts as the module reads them: bootstrap
// converts project's PlanningFacts into them.
type ProjectFacts = app.ProjectFacts

// IssueTargets is what the module tells the issue module about the cycles
// and modules of a project whose row lock the asking transaction holds (M6
// P3a spec R1, R9): the project's toggles and, of the cycles and modules a
// work item's write names, the project's undeleted ones with their names
// and archives; whether a cycle has ended on the day now falls on in the
// project's time zone; and, on that day, the ids of the cycles that have
// not ended and of the modules whose target day has not passed. bootstrap
// converts it into issue's PlanningPorts.
type IssueTargets interface {
	Targets(ctx context.Context, project uuid.UUID, cycles, modules []uuid.UUID) (IssueTargetFacts, error)
	CycleCompleted(ctx context.Context, project, cycle uuid.UUID, now time.Time) (bool, error)
	OpenTargets(ctx context.Context, project uuid.UUID, now time.Time) (cycles, modules []uuid.UUID, err error)
}

// IssueTargetFacts and IssueTarget are the values IssueTargets hands over:
// bootstrap converts them into issue's.
type (
	IssueTargetFacts = app.IssueTargetFacts
	IssueTarget      = app.IssueTarget
)

// IssueDirectory is what the module tells the issue module about the
// cycles and modules of the projects a list of work items ordered by their
// names reads (M6 P3b spec R2): of the projects, the names of the undeleted
// cycles and of the undeleted modules, archived or not, in id order, read in
// the list's read-only snapshot whatever the projects' toggles. bootstrap
// converts it into issue's PlanningPorts.
type IssueDirectory interface {
	CycleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error)
	ModuleNames(ctx context.Context, projects []uuid.UUID) ([]IssueDirectoryName, error)
}

// IssueDirectoryName is the value IssueDirectory hands over: bootstrap
// converts it into issue's.
type IssueDirectoryName = app.IssueDirectoryName

// IssueStats is what the module asks the issue module about the work items
// of a project's cycles and modules, which their progress and their list
// rows are computed from (M6 P4 spec R2): bootstrap converts issue's
// PlanningStats into it.
type IssueStats = app.IssueStats

// WorkItem, StateCount and CycleAssignee are the values IssueStats hands
// over, and StateGroup a state's group, which Projects' StateGroups hands
// over: bootstrap converts issue's and project's into them.
type (
	WorkItem      = domain.WorkItem
	StateCount    = domain.StateCount
	CycleAssignee = domain.CycleAssignee
	StateGroup    = domain.StateGroup
)

// Provided are the adapters bootstrap builds before wiring the modules.
type Provided struct {
	Purger         *app.Purger
	IssueTargets   IssueTargets
	IssueDirectory IssueDirectory
}

// Provide builds, over pool, planning's purge: its tables, children before
// parents, so no parent's CASCADE takes a child's rows first (M6 P2 spec
// 2.4, P3b spec 2.5); its issue targets, which read a project's facts through projects,
// project's PlanningProjects converted; and its issue directory. Neither
// the purge nor the directory uses projects.
func Provide(pool *pgxpool.Pool, projects app.FactsReader) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{
			{Table: "module_user_properties", Delete: store.PurgeModuleUserProperties},
			{Table: "module_links", Delete: store.PurgeModuleLinks},
			{Table: "module_members", Delete: store.PurgeModuleMembers},
			{Table: "modules", Delete: store.PurgeModules},
			{Table: "cycle_user_properties", Delete: store.PurgeCycleUserProperties},
			{Table: "cycles", Delete: store.PurgeCycles},
		}),
		IssueTargets:   app.NewIssueTargets(projects, store),
		IssueDirectory: app.NewIssueDirectory(store),
	}
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
	// Events publishes a cycle's and a module's deletion in the deleting
	// transaction: bootstrap's dispatcher, whose issue subscriber takes the
	// work items out of the deleted cycle or module (M6 P3a spec R6).
	Events shared.EventPublisher
	// Projects are project's PlanningProjects, converted (bootstrap/ports.go).
	Projects app.Projects
	// IssueStats are the issue module's PlanningStats, converted
	// (bootstrap/planning_issues.go): the facts of the work items of the
	// project's cycles and modules (M6 P4 spec R2).
	IssueStats app.IssueStats
	// ListSettings checks the words of the pages' display settings: the
	// issue module's ListSettings, as issue.Provide offers it (M6 P3b spec
	// R5).
	ListSettings app.IssueListSettings
	// LinkCrawls inserts the module links' crawls, planning.crawl_link, in
	// the transaction of the write that asks for them, which it requires:
	// bootstrap's jobInserter (M6 P2 spec R11).
	LinkCrawls riveradapter.Inserter
	// LinkCrawler is the link-preview crawler every module that has links
	// shares, adapted by bootstrap (R12).
	LinkCrawler app.LinkCrawler
	// JobTx runs the deletion job's and the freeze job's short
	// transactions: each lock wait at most issue.write_lock_timeout, no
	// write gate, and a lock failure kept as PostgreSQL's for River's
	// retry (bootstrap/gate.go).
	JobTx shared.TxManager
	// RuntimeNow is the monotonic clock of the deletion job's budget.
	RuntimeNow   func() time.Time
	DeleteBatch  int
	DeleteBudget time.Duration
	// FreezeInterval is the freeze job's: it runs when the server starts,
	// then every FreezeInterval (planning.freeze_interval).
	FreezeInterval time.Duration
	Logger         *slog.Logger
}

// Module is the planning module.
type Module struct {
	uc             httpadapter.UseCases
	crawlLink      *app.CrawlLink
	deleteRows     *app.DeleteRows
	deleteBudget   time.Duration
	freezeCycles   *app.FreezeCycles
	freezeInterval time.Duration
	logger         *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.WriteGate, d.Tx)
	reads := app.NewReads(d.Projects, store, d.Authorizer, d.ReadTx)
	links := app.ModuleLinkDeps{Locks: locks, Links: store, Crawls: riveradapter.NewLinkCrawls(d.LinkCrawls), Clock: d.Clock}
	pageReads := app.GetPreferencesDeps{Reads: reads, Preferences: store}
	pageWrites := app.UpdatePreferencesDeps{Locks: locks, Settings: d.ListSettings, Clock: d.Clock, Preferences: store}
	return &Module{
		uc: httpadapter.UseCases{
			ListCycles: app.NewListCycles(app.ListCyclesDeps{Reads: reads, Cycles: store, Groups: d.Projects, Stats: d.IssueStats, Snapshots: store,
				Clock: d.Clock}),
			CreateCycle:      app.NewCreateCycle(app.CreateCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			GetCycle:         app.NewGetCycle(app.GetCycleDeps{Reads: reads, Clock: d.Clock}),
			UpdateCycle:      app.NewUpdateCycle(app.UpdateCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			ArchiveCycle:     app.NewArchiveCycle(app.ArchiveCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			UnarchiveCycle:   app.NewUnarchiveCycle(app.ArchiveCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock}),
			DeleteCycle:      app.NewDeleteCycle(app.DeleteCycleDeps{Locks: locks, Cycles: store, Clock: d.Clock, Events: d.Events}),
			ListModules:      app.NewListModules(app.ListModulesDeps{Reads: reads, Modules: store, Groups: d.Projects, Stats: d.IssueStats}),
			CreateModule:     app.NewCreateModule(app.CreateModuleDeps{Locks: locks, Roles: d.Projects, Modules: store, Clock: d.Clock}),
			GetModule:        app.NewGetModule(app.GetModuleDeps{Reads: reads}),
			UpdateModule:     app.NewUpdateModule(app.UpdateModuleDeps{Locks: locks, Roles: d.Projects, Modules: store, Clock: d.Clock}),
			ArchiveModule:    app.NewArchiveModule(app.ArchiveModuleDeps{Locks: locks, Modules: store, Clock: d.Clock}),
			UnarchiveModule:  app.NewUnarchiveModule(app.ArchiveModuleDeps{Locks: locks, Modules: store, Clock: d.Clock}),
			DeleteModule:     app.NewDeleteModule(app.DeleteModuleDeps{Locks: locks, Modules: store, Clock: d.Clock, Events: d.Events}),
			CreateModuleLink: app.NewCreateModuleLink(links),
			UpdateModuleLink: app.NewUpdateModuleLink(links),
			DeleteModuleLink: app.NewDeleteModuleLink(links),
			ListModuleLinks:  app.NewListModuleLinks(app.ListModuleLinksDeps{Reads: reads, Links: store}),

			GetCycleProgress: app.NewGetCycleProgress(app.GetCycleProgressDeps{Reads: reads, Groups: d.Projects, Stats: d.IssueStats,
				Snapshots: store, Clock: d.Clock}),
			GetModuleProgress: app.NewGetModuleProgress(app.GetModuleProgressDeps{Reads: reads, Groups: d.Projects, Stats: d.IssueStats,
				Clock: d.Clock}),

			GetCyclePreferences:     app.NewGetCyclePreferences(pageReads),
			UpdateCyclePreferences:  app.NewUpdateCyclePreferences(pageWrites),
			GetModulePreferences:    app.NewGetModulePreferences(pageReads),
			UpdateModulePreferences: app.NewUpdateModulePreferences(pageWrites),
		},
		crawlLink:    app.NewCrawlLink(app.CrawlLinkDeps{Crawler: d.LinkCrawler, Links: store}),
		deleteRows:   app.NewDeleteRows(app.DeleteRowsDeps{Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow, Batch: d.DeleteBatch, Budget: d.DeleteBudget}),
		deleteBudget: d.DeleteBudget, logger: d.Logger,
		freezeCycles: app.NewFreezeCycles(app.FreezeCyclesDeps{Projects: d.Projects, Groups: d.Projects, Cycles: store, Stats: d.IssueStats,
			Tx: d.JobTx, Clock: d.Clock}),
		freezeInterval: d.FreezeInterval,
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
