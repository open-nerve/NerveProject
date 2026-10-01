// Package app holds the project module's use cases (M3 design 6.3) and the
// ports they need: small repository interfaces per use case, the clock, and
// the other modules' adapters as bootstrap converts them.
package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Clock is the use cases' time.
type Clock interface {
	Now() time.Time
}

// Workspace is a workspace as WorkspaceDirectory finds it: bootstrap
// converts the workspace module's answer into it (M3 design 6.5).
type Workspace struct {
	ID       uuid.UUID
	Timezone string
}

// WorkspaceDirectory is the workspace module's directory
// (workspace.Provide): the undeleted workspace a slug names, or an id
// names; found is false when there is none (M3 design 6.5).
type WorkspaceDirectory interface {
	// WorkspaceBySlug reads it without a lock: for a read.
	WorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
	// ShareWorkspaceBySlug also locks the workspace's row FOR SHARE until
	// the transaction ctx carries ends: the parent lock of a write that adds
	// a project (M3 design 3.6 convention 2). A workspace deleted while the
	// lock waited is not found.
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
	WorkspaceSharer
}

// WorkspaceSharer takes the first lock of a write on a project (M3 design
// 3.6 convention 2).
type WorkspaceSharer interface {
	// ShareWorkspaceByID locks the undeleted workspace id's row FOR SHARE
	// until the transaction ctx carries ends. Every cascade over the
	// workspace's projects runs under the row's FOR NO KEY UPDATE, which
	// this waits for and holds off; another write on a project of the
	// workspace does not wait for it. A workspace deleted while the lock
	// waited is not found.
	ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (w Workspace, found bool, err error)
}

// WorkspaceMembers is the workspace module's lock of the memberships a
// write makes project members from (workspace.Provide; M3 design 3.6
// convention 3).
type WorkspaceMembers interface {
	// ShareMembers locks userIDs' undeleted memberships of workspaceID,
	// active or not, FOR SHARE in id order until the transaction ctx
	// carries ends, and returns the roles of the active ones by account.
	ShareMembers(ctx context.Context, workspaceID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// ProjectReader reads a project as a user sees it.
type ProjectReader interface {
	// GetProject is the undeleted project id as userID sees it; found is
	// false when there is none.
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
}

// ProjectLister is listProjects' repository.
type ProjectLister interface {
	// ListProjects lists workspaceID's undeleted projects that userID sees
	// with v, the archived ones alone when archived is true and the others
	// otherwise, each as userID sees it: by its place in userID's sidebar,
	// the projects without a place in it last, then by name (M3 design
	// 3.12).
	ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error)
}

// IdentifierReader is checkProjectIdentifier's repository.
type IdentifierReader interface {
	// IdentifierTaken reports whether an undeleted project of workspaceID
	// has identifier, compared as stored, in upper case: the caller
	// upper-cases it.
	IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error)
}

// ProjectCreator is createProject's repository. Each method runs in the
// transaction ctx carries.
type ProjectCreator interface {
	ProjectReader
	CreateProject(ctx context.Context, p ProjectRow) error
	CreateMember(ctx context.Context, m MemberRow) error
	// LowestSortOrder is the least place of userID's in his sidebar among
	// workspaceID's projects, nil when he has none.
	LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error)
	CreatePreferences(ctx context.Context, p PreferencesRow) error
	CreateStates(ctx context.Context, rows []StateRow) error
}

// LockedProject is a project as its lock reads it.
type LockedProject struct {
	WorkspaceID uuid.UUID
	Archived    bool
}

// ProjectLocker takes the parent lock of a write on a project (M3 design
// 3.6 convention 2).
type ProjectLocker interface {
	// LockProject locks the undeleted project id FOR NO KEY UPDATE until the
	// transaction ctx carries ends; found is false when there is none, a
	// project deleted while the lock waited too.
	LockProject(ctx context.Context, id uuid.UUID) (p LockedProject, found bool, err error)
}

// ProjectFinder finds the workspace of a project: what a read decides on,
// and what a write on the project reads first, to lock the workspace
// before the project (M3 design 3.6 convention 2).
type ProjectFinder interface {
	// ProjectWorkspace is the workspace of the undeleted project id, read
	// without a lock; found is false when there is none.
	ProjectWorkspace(ctx context.Context, id uuid.UUID) (workspaceID uuid.UUID, found bool, err error)
}

// Membership is an account's undeleted membership of a project, active or
// ended.
type Membership struct {
	ID     uuid.UUID
	Role   shared.Role
	Active bool
}

// MembershipReader reads an account's membership of a project.
type MembershipReader interface {
	// Memberships are userIDs' undeleted memberships of projectID, active or
	// ended, by account; an account without one is not in it. Under the
	// project's FOR NO KEY UPDATE they stay as read.
	Memberships(ctx context.Context, projectID uuid.UUID, userIDs []uuid.UUID) (map[uuid.UUID]Membership, error)
}

// ProjectUpdater is updateProject's repository. Each method runs in the
// transaction ctx carries.
type ProjectUpdater interface {
	ProjectReader
	ProjectLocker
	MembershipReader
	// UpdateProject changes the fields p gives of the project id, by the
	// account by at now. An identifier or a name another undeleted project of
	// the workspace has is domain.ErrIdentifierTaken or domain.ErrNameTaken.
	UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error
}

// ProjectRow is a project to insert: checked values, its id, its creator
// and the time of the use case's clock. Every column it does not name takes
// its default.
type ProjectRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Description string
	Identifier  string
	Network     domain.Network
	LeadID      *uuid.UUID
	LogoProps   domain.LogoProps
	Timezone    string
	CreatedBy   uuid.UUID
	Now         time.Time
}

// MemberRow is a project membership to insert, active.
type MemberRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// PreferencesRow is an account's display settings in a project to insert:
// the default navigation, and the place in his sidebar given (M3 design
// 3.18).
type PreferencesRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	UserID      uuid.UUID
	SortOrder   float64
	CreatedBy   uuid.UUID
	Now         time.Time
}

// StateRow is a state to insert.
type StateRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	State       domain.NewState
	CreatedBy   uuid.UUID
	Now         time.Time
}

// MemberDemoter makes an account a guest in a workspace's projects (M3
// design 3.3, 3.6): his projects locked first, then his memberships of them
// in one statement. Each method runs in the transaction ctx carries.
type MemberDemoter interface {
	// LockMemberProjects locks FOR NO KEY UPDATE, in id order, workspaceID's
	// undeleted projects in which userID has an undeleted membership, active
	// or not, and returns their ids.
	LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error)
	// DemoteMemberships sets role 5, updated_at now and updated_by_id by on
	// userID's undeleted memberships of projectIDs, active or not, that are
	// not a guest's already.
	DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error
}

// WorkspaceProjectsDeleter soft-deletes a workspace's projects and the rows
// under them, one statement a table, in the order of M3 design 3.6. Each
// sets deleted_at and updated_at to now and updated_by_id to by, on the
// workspace's undeleted rows only; it runs in the transaction ctx carries.
type WorkspaceProjectsDeleter interface {
	DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceProjectMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceProjectPreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	DeleteWorkspaceStates(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}
