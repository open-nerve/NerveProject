// Package project is the projects module (M3 design 3.3, 6.3): projects,
// their members, their states, their labels and each member's display
// settings. It brings listing, creating, reading, changing, archiving and
// deleting projects, checking an identifier, listing, adding and joining
// the members, changing a member's role, removing a member and leaving,
// each member's display settings, listing, creating, changing and deleting
// a project's states and making one its default, listing the states of a
// workspace's projects one is a member of, listing, creating, changing and
// deleting a project's labels, carries out the workspace module's cascades
// on the projects (ProjectCascade), and offers the access module its reads
// of a project (ProjectAccess), the workspace module its count of an
// account's ended project memberships (ProjectMembershipCounts), and the
// issue module the project's lock, numbers and catalogs (IssueProjects).
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
	// LockWorkspaceProjects locks the workspace's undeleted projects FOR NO
	// KEY UPDATE in id order and returns their ids: deleteWorkspace's lock
	// of its projects, before it reads its moment (M4 design 3.5).
	LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error)
	// DeleteWorkspaceProjects soft-deletes the workspace's projects and the
	// rows under them, the work-item side of projectIDs, those it locked,
	// too: deleteWorkspace's last step.
	DeleteWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error
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
	// Issues is issue's IssueCascade (issue.Provide): deleting a
	// workspace's projects deletes their work-item side (M4 design 6.6).
	Issues app.IssueCascade
}

// NewCascade builds the module's Cascade alone, on the pool: for the
// command line, whose `nerve users deactivate` ends memberships through it
// and builds no HTTP side (M3 design 6.3, 6.6). New builds its own through
// it.
func NewCascade(d CascadeDeps) Cascade {
	store := postgresadapter.New(d.Pool)
	return app.NewCascade(store, store, store, store, d.Issues)
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

// IssueProjects is what the issue module asks of a project (M4 design
// 6.5): its gate (ProjectGate's Find, Lock and LockMany), its work items'
// numbers (Sequences), and its states', labels' and members' catalogs
// (StateCatalog, LabelCatalog, ProjectMembers), for its writes; a
// workspace's projects' identifiers (ProjectIdentifiers), for a read by
// identifier; ordinary and historical memberships; and complete
// state and label catalogs for lists. Each reads, or locks, in the transaction
// ctx carries. Write catalogs are read under the project's lock, which every
// write of a state, a label or a membership holds too, so they stay as
// read until the write commits (M4 design 3.5).
type IssueProjects interface {
	// DeletedIssueProjects reads real deletion times without locking; absent IDs are omitted.
	DeletedIssueProjects(context.Context, []uuid.UUID) (map[uuid.UUID]time.Time, error)
	// AllUndeletedIssueProjectIDs is the complete retention directory, in id order.
	AllUndeletedIssueProjectIDs(context.Context) ([]uuid.UUID, error)
	// UndeletedIssueProjectIDs includes archived projects and ignores caller membership.
	UndeletedIssueProjectIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	// ShareIssueProjects locks the undeleted rows once, ascending, FOR SHARE.
	ShareIssueProjects(context.Context, []uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	// VisibleIssueProjects reads active, undeleted memberships in undeleted,
	// unarchived projects, using the project role for guest scope.
	VisibleIssueProjects(ctx context.Context, workspaceID, actorID uuid.UUID) ([]VisibleIssueProject, error)
	// IssueListStates reads complete undeleted non-triage state catalogs.
	IssueListStates(ctx context.Context, projectIDs []uuid.UUID) ([]IssueState, error)
	// IssueListLabels reads complete undeleted label catalogs.
	IssueListLabels(ctx context.Context, projectIDs []uuid.UUID) ([]IssueLabel, error)
	// HistoricalIssueProjects reads active memberships, including deleted or archived projects.
	HistoricalIssueProjects(ctx context.Context, workspaceID, actorID uuid.UUID) ([]HistoricalIssueProject, error)
	// IssueMentionMembers reads active candidates and their effective guest scope.
	IssueMentionMembers(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]IssueMemberVisibility, error)
	// ProjectWorkspace is the workspace of the undeleted project id,
	// archived or not, read without a lock; found is false when there is
	// none.
	ProjectWorkspace(ctx context.Context, id uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
	// LockIssueProject locks the undeleted project id FOR NO KEY UPDATE,
	// its workspace not locked, and answers it; found is false when there
	// is none, also when it was deleted while the lock waited.
	LockIssueProject(ctx context.Context, id uuid.UUID) (p IssueProject, found bool, err error)
	// LockIssueProjects locks the undeleted projects among ids so, in one
	// statement, in id order, and answers them by id; one deleted while
	// the lock waited is not among them.
	LockIssueProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]IssueProject, error)
	// NextIssueSequence takes the project's next work-item number.
	NextIssueSequence(ctx context.Context, id uuid.UUID) (int, error)
	// IssueStates reads the project's undeleted states, its triage state
	// too, in id order.
	IssueStates(ctx context.Context, projectID uuid.UUID) ([]IssueState, error)
	// IssueLabels reads those of ids that are the project's undeleted
	// labels, in id order.
	IssueLabels(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) ([]IssueLabel, error)
	// ActiveRoles reads the project roles of those of the accounts ids
	// that are its active members, by account.
	ActiveRoles(ctx context.Context, projectID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
	// ProjectIdentifiers reads the workspace's undeleted projects, archived
	// or not, each with its identifier, in id order, without a lock.
	ProjectIdentifiers(ctx context.Context, workspaceID uuid.UUID) ([]IssueProjectIdentifier, error)
}

// IssueProject, IssueState, IssueLabel and IssueProjectIdentifier are the
// values IssueProjects hands over: bootstrap converts them into issue's
// (M4 design 6.5).
type (
	HistoricalIssueProject = app.HistoricalIssueProject
	VisibleIssueProject    = app.VisibleIssueProject
	IssueMemberVisibility  = app.IssueMemberVisibility
	IssueProject           = app.IssueProject
	IssueState             = app.IssueState
	IssueLabel             = app.IssueLabel
	IssueProjectIdentifier = app.IssueProjectIdentifier
)

// Provided are the adapters project offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	Purger                  *app.Purger
	ArchivePolicies         ArchivePolicies
	ProjectAccess           ProjectAccess
	ProjectMembershipCounts ProjectMembershipCounts
	IssueProjects           IssueProjects
}

// Provide builds project's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{
		Purger: app.NewPurger([]app.PurgeStep{
			{Table: "project_user_properties", Delete: store.PurgeProjectUserProperties},
			{Table: "project_members", Delete: store.PurgeProjectMembers},
			{Table: "labels", Delete: store.PurgeLabels},
			{Table: "states", Delete: store.PurgeStates},
			{Table: "projects", Delete: store.PurgeProjects},
		}),
		ArchivePolicies: store, ProjectAccess: store,
		ProjectMembershipCounts: store, IssueProjects: store,
	}
}

// Workspace is a workspace as the WorkspaceDirectory port hands it over:
// bootstrap converts workspace's DirectoryEntry into it (M3 design 6.5).
type Workspace = app.Workspace

// ListCalendar and ListDisplayFilters are the settings consumer boundary values.
type ListCalendar = app.ListCalendar
type ListDisplayFilters = app.ListDisplayFilters

// Deps are what bootstrap gives the module (M3 design 6.6, step 4).
type Deps struct {
	Events     shared.EventPublisher
	Pool       *pgxpool.Pool
	Tx         shared.TxManager
	Clock      app.Clock
	Authorizer shared.Authorizer
	// Workspaces is workspace's WorkspaceDirectory, converted
	// (bootstrap/ports.go).
	Workspaces app.WorkspaceDirectory
	// Members is workspace's WorkspaceMembers (workspace.Provide).
	Members app.WorkspaceMembers
	// Issues is issue's IssueCascade (issue.Provide).
	Issues app.IssueCascade
	// ListSettings is the pure issue-list vocabulary validator, converted by bootstrap.
	ListSettings app.IssueListSettings
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
	return &Module{cascade: NewCascade(CascadeDeps{Pool: d.Pool, Issues: d.Issues}), uc: httpadapter.UseCases{
		CreateProject: app.NewCreateProject(app.CreateProjectDeps{
			Workspaces: d.Workspaces, Members: d.Members, Projects: store, Auth: d.Authorizer, Tx: d.Tx, Clock: d.Clock,
		}),
		ListProjects:        app.NewListProjects(d.Workspaces, store, d.Authorizer),
		GetProject:          app.NewGetProject(store, d.Authorizer),
		CheckIdentifier:     app.NewCheckProjectIdentifier(d.Workspaces, store, d.Authorizer),
		UpdateProject:       app.NewUpdateProject(store, locks, d.Tx, d.Clock),
		ArchiveProject:      app.NewArchiveProject(store, locks, d.Tx, d.Clock),
		UnarchiveProject:    app.NewUnarchiveProject(store, locks, d.Tx, d.Clock),
		DeleteProject:       app.NewDeleteProject(store, d.Events, locks, d.Tx, d.Clock),
		GetPreferences:      app.NewGetProjectPreferences(store, d.Authorizer),
		UpdatePreferences:   app.NewUpdateProjectPreferences(store, locks, d.Tx, d.Clock, d.ListSettings),
		ListMembers:         app.NewListProjectMembers(store, d.Authorizer),
		AddMembers:          app.NewAddProjectMembers(app.AddMembersDeps{Locks: locks, Projects: store, Tx: d.Tx, Clock: d.Clock}),
		JoinProject:         app.NewJoinProject(locks, store, d.Tx, d.Clock),
		UpdateMember:        app.NewUpdateProjectMember(locks, store, d.Tx, d.Clock),
		RemoveMember:        app.NewRemoveProjectMember(locks, store, d.Tx, d.Clock),
		LeaveProject:        app.NewLeaveProject(locks, store, d.Tx, d.Clock),
		ListStates:          app.NewListStates(store, d.Authorizer),
		CreateState:         app.NewCreateState(locks, store, d.Tx, d.Clock),
		UpdateState:         app.NewUpdateState(locks, store, d.Tx, d.Clock),
		DeleteState:         app.NewDeleteState(locks, store, d.Issues, d.Tx, d.Clock),
		MarkDefaultState:    app.NewMarkDefaultState(locks, store, d.Tx, d.Clock),
		ListWorkspaceStates: app.NewListWorkspaceStates(d.Workspaces, store, d.Authorizer),
		ListLabels:          app.NewListLabels(store, d.Authorizer),
		CreateLabel:         app.NewCreateLabel(locks, store, d.Tx, d.Clock),
		UpdateLabel:         app.NewUpdateLabel(locks, store, d.Tx, d.Clock),
		DeleteLabel:         app.NewDeleteLabel(locks, store, d.Issues, d.Tx, d.Clock),
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
