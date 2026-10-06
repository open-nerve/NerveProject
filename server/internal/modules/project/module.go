// Package project is the projects module (M3 design 3.3, 6.3): projects,
// their members, their states, their labels and each member's display
// settings. It brings listing, creating, reading, changing, archiving and
// deleting projects, checking an identifier, listing, adding and joining
// the members, changing a member's role, removing a member and leaving,
// each member's display settings, listing, creating, changing and deleting
// a project's states and making one its default, listing the states of a
// workspace's projects one is a member of, creating, changing and deleting
// a project's labels, carries out the workspace module's cascades on the
// projects (ProjectCascade), and offers the access module its reads of a
// project (ProjectAccess) and the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts).
package project

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Cascade is what the workspace module's writes ask of the projects:
// workspace's ProjectCascade port (M3 design 3.3). Each method runs in the
// caller's transaction, which ctx carries, writes by and now into the rows
// it changes, and returns a failure as itself, so the caller's whole write
// rolls back.
type Cascade interface {
	// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
	// rows under them: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	// DemoteToGuest makes userID a guest in each of the workspace's projects
	// he has a membership of, ended ones too.
	DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error
	// EndMemberships ends userID's active memberships of the workspaces'
	// projects, found when it is called and locked in the projects' id
	// order across the workspaces; project.sole_admin, and nothing ended,
	// when he is the only active admin of one that has other active members.
	EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// CascadeDeps are what a composition that wants the cascade alone gives
// it (M3 design 6.3, 6.6).
type CascadeDeps struct {
	Pool *pgxpool.Pool
}

// NewCascade builds the module's Cascade alone, on the pool: for the
// command line, whose `nerve users deactivate` ends memberships through it
// and builds no HTTP side (M3 design 6.3, 6.6). New builds its own through
// it.
func NewCascade(d CascadeDeps) Cascade {
	store := postgresadapter.New(d.Pool)
	return app.NewCascade(store, store, store)
}

// ProjectAccess reads a project for the access module's decision (M3
// design 6.5): the facts of the undeleted project projectID, archived or
// not, for userID; found is false for a project that does not exist or is
// deleted. It reads in the transaction ctx carries.
type ProjectAccess interface {
	ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f AccessFacts, found bool, err error)
}

// AccessFacts are the facts ProjectAccess hands over: bootstrap converts
// them into access's value (M3 design 6.5).
type AccessFacts = app.AccessFacts

// ProjectMembershipCounts counts an account's memberships of a workspace's
// projects, for the workspace module's reactivate-member (M3 design 3.11,
// 6.5): CountInactive is the number of userID's ended, undeleted
// memberships of the workspace's projects, read without a lock.
type ProjectMembershipCounts interface {
	CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error)
}

// Provided are the adapters project offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	ProjectAccess           ProjectAccess
	ProjectMembershipCounts ProjectMembershipCounts
}

// Provide builds project's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{ProjectAccess: store, ProjectMembershipCounts: store}
}

// Workspace is a workspace as the WorkspaceDirectory port hands it over:
// bootstrap converts workspace's DirectoryEntry into it (M3 design 6.5).
type Workspace = app.Workspace

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
type Deps struct {
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Workspaces is workspace's WorkspaceDirectory, converted
	// (bootstrap/ports.go).
	Workspaces app.WorkspaceDirectory
	// Members is workspace's WorkspaceMembers (workspace.Provide).
	Members app.WorkspaceMembers
}

// Module is the wired project module.
type Module struct {
	cascade Cascade
	uc      httpadapter.UseCases
}

// New wires the module's use cases and its HTTP side from d; nothing is
// registered or injected after it (M3 design 6.6). bootstrap builds it
// before workspace, which takes its Cascade.
func New(d Deps) *Module {
	store := postgresadapter.New(d.Pool)
	locks := app.NewLocks(store, d.Workspaces, d.Members, d.Authorizer)
	return &Module{cascade: NewCascade(CascadeDeps{Pool: d.Pool}), uc: httpadapter.UseCases{
		CreateProject: app.NewCreateProject(app.CreateProjectDeps{
			Workspaces: d.Workspaces, Members: d.Members, Projects: store, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock,
		}),
		ListProjects:        app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:          app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:     app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:       app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:      app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject:    app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:       app.NewDeleteProject(store, locks, d.Tx, d.Clock),
		GetPreferences:      app.NewGetProjectPreferences(store, d.Authorizer),
		UpdatePreferences:   app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock),
		ListMembers:         app.NewListProjectMembers(store, d.Authorizer),
		AddMembers:          app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
		JoinProject:         app.NewJoinProject(locks, store, d.Tx, d.Clock),
		UpdateMember:        app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
		RemoveMember:        app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
		LeaveProject:        app.NewLeaveProject(locks, store, d.Tx, d.Clock),
		ListStates:          app.NewListStates(store, d.Authorizer),
		CreateState:         app.NewCreateState(locks, store, d.Tx, d.Clock),
		UpdateState:         app.NewUpdateState(locks, store, d.Tx, d.Clock),
		DeleteState:         app.NewDeleteState(locks, store, d.Tx, d.Clock),
		MarkDefaultState:    app.NewMarkDefaultState(locks, store, d.Tx, d.Clock),
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
		CreateLabel:         app.NewCreateLabel(locks, store, d.Tx, d.Clock),
		UpdateLabel:         app.NewUpdateLabel(locks, store, d.Tx, d.Clock),
		DeleteLabel:         app.NewDeleteLabel(locks, store, d.Tx, d.Clock),
	}}
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc)
}

// Cascade is the module's implementation of workspace's ProjectCascade.
func (m *Module) Cascade() Cascade {
	return m.cascade
}

// Actions lists the module's actions: bootstrap's test holds the union of
// every module's actions equal to access's rule table (M3 design 3.4).
func Actions() []shared.Action {
	return domain.Actions()
}
