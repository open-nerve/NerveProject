// Package app holds the project module's use cases (M3 design 6.3) and the
// ports they need: small repository interfaces per use case, the clock, and
// the other modules' adapters as bootstrap converts them.
package app

import (
	"context"
	"time"
	"uuid"
)

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
