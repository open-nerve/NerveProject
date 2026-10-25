package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// VisibleProject is an actor's project membership and guest scope.
type VisibleProject struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}

// VisibleProjects reads ordinary and historical membership scopes in the read
// transaction carried by ctx.
type VisibleProjects interface {
	// Visible requires active, undeleted membership in an undeleted,
	// unarchived project. Workspace roles do not widen project guest scope.
	Visible(ctx context.Context, workspaceID, actorID uuid.UUID) ([]VisibleProject, error)
	// Historical retains active memberships despite project/member soft deletion
	// or archival. OwnOnly follows the project role and guest visibility flag.
	Historical(ctx context.Context, workspaceID, actorID uuid.UUID) ([]VisibleProject, error)
}
