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
