// Package issue is the work items module (M4 design 3.3, 6.2): it owns the
// work items' tables and brings creating a work item. For the project
// module's deletions (its IssueCascade port) it soft-deletes a deleted
// project's rows of them, tells whether work items are in a state, and
// takes deleted labels off them.
package issue

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deleteBatch is the rows a statement of a project's deletion soft-deletes
// at most (M4 design 3.6).
const deleteBatch = 1000

// Cascade is what the project module's deletions ask of the work items:
// project's IssueCascade port (M4 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, and returns a failure as
// itself, so the caller's whole write rolls back; one that deletes writes
// by and now into the rows it deletes.
type Cascade interface {
	// DeleteProjectIssues soft-deletes the project's work-item-side rows.
	DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceIssues soft-deletes the work-item-side rows of each
	// of the projects.
	DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
	// StateInUse reports whether an undeleted work item, archived or not,
	// is in the state (M4 design 3.6).
	StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error)
	// DetachLabel soft-deletes the rows that put the labels on work items.
	DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error
}

// Provided are the adapters issue offers the other modules. They depend on
// the pool alone, so bootstrap builds them before any module (M4 design
// 6.6).
type Provided struct {
	IssueCascade Cascade
}

// Provide builds issue's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{IssueCascade: app.NewCascade(store, store, deleteBatch)}
}

// ProjectPorts are what the module's writes ask of a project (M4 design
// 6.5): its gate, its numbers, its states', labels' and members' catalogs.
// project.Provide's IssueProjects is them, converted (bootstrap/ports.go).
type ProjectPorts interface {
	app.ProjectGate
	app.Sequences
	app.StateCatalog
	app.LabelCatalog
	app.ProjectMembers
}

// LockedProject, StateRef and its StateGroup, and LabelRef are the values
// ProjectPorts hand over: bootstrap converts project's into them (M4
// design 6.5).
type (
	LockedProject = app.LockedProject
	StateRef      = app.StateRef
	StateGroup    = domain.StateGroup
	LabelRef      = app.LabelRef
)

// Deps are what bootstrap gives the module (M4 design 6.6, step 6).
type Deps struct {
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Projects are project's IssueProjects, converted (bootstrap/ports.go).
	Projects ProjectPorts
	// Profiles is identity's PublicProfiles, converted (bootstrap/ports.go).
	Profiles app.PublicProfiles
}

// Module is the wired issue module.
type Module struct {
	uc httpadapter.UseCases
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M4 design 6.6).
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(d.Projects, store, d.Authorizer, d.Tx)
	return &Module{uc: httpadapter.UseCases{
		CreateIssue: app.NewCreateIssue(app.CreateIssueDeps{
			Locks: locks, Issues: store, Sequences: d.Projects, States: d.Projects, Labels: d.Projects, Members: d.Projects,
			Profiles: d.Profiles, Clock: d.Clock,
		}),
		GetIssue: app.NewGetIssue(locks),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
