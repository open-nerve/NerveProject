// Package notification is the notifications module (M7 design 3.2–3.4, 4):
// it owns the notifications table, each row one account's notification of
// one change of a work item, with the snapshot of the work item and the
// change at the moment it was sent. The fan-out sends an event's
// notifications under the work item's row lock (M7 P1b spec 2.9). A work
// item's deletion deletes its notifications in its transaction; a project's
// or a workspace's deletion enqueues the deletion of each deleted project's
// notifications, which waits for the project's work items to be deleted
// (spec 2.8). The 60-day purge removes the deleted ones (Provide). Its API
// lists a receiver's notifications in a workspace, counts his unread ones,
// marks them read, and changes one's states, each as he reads its project
// and its work item now (M7 P1c spec 2).
package notification

import (
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
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

// Issues are what the module asks of the issue module (spec 2.6): whether
// a project has an undeleted work item; a work item's row lock, the reads
// of it the fan-out makes after the lock, and the guest rule its recheck
// applies.
type Issues interface {
	app.ProjectIssues
	app.IssueRowLock
	app.IssueReads
	app.GuestRule
}

// LockedIssue, IssueColumns, StateName, Decision, Grant and IssueRef are
// the values the module's ports answer or take, which bootstrap converts
// the issue, the project and the access modules' into and out of.
type (
	LockedIssue  = app.LockedIssue
	IssueColumns = app.IssueColumns
	StateName    = app.StateName
	Decision     = domain.Decision
	Grant        = app.Grant
	IssueRef     = domain.IssueRef
)

// Deps are what the module needs.
type Deps struct {
	Pool *pgxpool.Pool
	// Issues is the issue module's NotificationIssues, converted
	// (bootstrap/notification_ports.go).
	Issues Issues
	// Projects are the project module's reads, without a lock, a
	// notification's snapshot is completed with, converted.
	Projects app.ProjectNames
	// Readers is the access module's Decider, converted: the recheck of
	// the recipients (spec 2.7).
	Readers app.Readers
	// JobTx runs the jobs' transactions, the deletion's short ones and the
	// fan-out's: each lock wait at most issue.write_lock_timeout, no write
	// gate, and a lock failure kept as PostgreSQL's for River's retry
	// (bootstrap/gate.go).
	JobTx shared.TxManager
	// Workspaces is the workspace module's directory, converted: a
	// workspace by its slug (P1c spec 2.2).
	Workspaces app.WorkspaceDirectory
	Authorizer shared.Authorizer
	// Readable is the access module's ReadableProjects, converted: the
	// projects a reader reads in a workspace now (spec 2.5, R1).
	Readable app.ReadableProjects
	// IssueRefs are the issue module's IssueRefs and the project module's
	// identifiers, composed and converted: the work items a page names, as
	// its reader reads them now (spec 2.6, R4).
	IssueRefs app.IssueRefs
	// Clock is the business clock (spec 2.3, 2.7, R6).
	Clock app.Clock
	// ReadTx supplies the reads' read-only snapshot transactions, each its
	// own: the list's, the unread counts' and mark-all-read's start.
	ReadTx shared.TxManager
	// Tx are the transactions of a change of a notification, which lock its
	// row: each lock wait at most issue.write_lock_timeout, answered
	// server_busy past it (bootstrap's busyTx, spec 2.7 step 4).
	Tx shared.TxManager
	// MarkTx are mark-all-read's batches' transactions, read-write, with no
	// lock wait of their own: a batch never waits for a notification's row
	// (SKIP LOCKED, R5).
	MarkTx shared.TxManager
	// RuntimeNow is the monotonic clock of the deletion job's budget and of
	// its wait for the project's work items.
	RuntimeNow   func() time.Time
	DeleteBatch  int
	DeleteBudget time.Duration
	Logger       *slog.Logger
}

// Module is the notification module.
type Module struct {
	uc           httpadapter.UseCases
	fanout       *app.Fanout
	deleteRows   *app.DeleteRows
	deleteBudget time.Duration
	logger       *slog.Logger
}

// New builds the module from d.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	inbox := app.InboxDeps{Workspaces: d.Workspaces, Authorizer: d.Authorizer, Readable: d.Readable, Clock: d.Clock}
	return &Module{
		uc: httpadapter.UseCases{
			ListNotifications: app.NewListNotifications(app.ListNotificationsDeps{InboxDeps: inbox, ReadTx: d.ReadTx, Lists: store, Refs: d.IssueRefs}),
			GetUnreadCount:    app.NewGetUnreadCount(app.GetUnreadCountDeps{InboxDeps: inbox, ReadTx: d.ReadTx, Lists: store}),
			MarkAllRead:       app.NewMarkAllRead(app.MarkAllReadDeps{InboxDeps: inbox, ReadTx: d.ReadTx, Tx: d.MarkTx, Marks: store}),
			UpdateNotification: app.NewUpdateNotification(app.UpdateNotificationDeps{Authorizer: d.Authorizer, Readable: d.Readable, GuestRule: d.Issues,
				Clock: d.Clock, Rows: store, Refs: d.IssueRefs, Tx: d.Tx}),
		},
		fanout: app.NewFanout(app.FanoutDeps{Lock: d.Issues, Issues: d.Issues, Projects: d.Projects, Readers: d.Readers,
			GuestRule: d.Issues, Rows: store, Tx: d.JobTx, Logger: d.Logger}),
		deleteRows: app.NewDeleteRows(app.DeleteRowsDeps{Issues: d.Issues, Rows: store, Tx: d.JobTx, RuntimeNow: d.RuntimeNow,
			Batch: d.DeleteBatch, Budget: d.DeleteBudget}),
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
