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
// (workspace.Provide): the undeleted workspace a slug names; found is false
// when there is none (M3 design 6.5).
type WorkspaceDirectory interface {
	// ShareWorkspaceBySlug also locks the workspace's row FOR SHARE until
	// the transaction ctx carries ends: the parent lock of a write that adds
	// a project (M3 design 3.6 convention 2). A workspace deleted while the
	// lock waited is not found.
	ShareWorkspaceBySlug(ctx context.Context, slug string) (w Workspace, found bool, err error)
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

// ProjectCreator is createProject's repository. Each method runs in the
// transaction ctx carries.
type ProjectCreator interface {
	CreateProject(ctx context.Context, p ProjectRow) error
	CreateMember(ctx context.Context, m MemberRow) error
	// LowestSortOrder is the least place of userID's in his sidebar among
	// workspaceID's projects, nil when he has none.
	LowestSortOrder(ctx context.Context, workspaceID, userID uuid.UUID) (*float64, error)
	CreatePreferences(ctx context.Context, p PreferencesRow) error
	CreateStates(ctx context.Context, rows []StateRow) error
	// GetProject is the undeleted project id as userID sees it; found is
	// false when there is none.
	GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error)
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
