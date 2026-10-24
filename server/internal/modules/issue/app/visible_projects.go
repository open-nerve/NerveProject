package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// VisibleProject is an actor's historical project membership and guest scope.
type VisibleProject struct {
	ID      uuid.UUID
	Role    shared.Role
	OwnOnly bool
}

// VisibleProjects reads historical memberships for filtering references after
// normal target authorization, in the same read transaction carried by ctx.
type VisibleProjects interface {
	// Historical retains active memberships despite project/member soft deletion
	// or archival. OwnOnly follows the project role and guest visibility flag.
	Historical(ctx context.Context, workspaceID, actorID uuid.UUID) ([]VisibleProject, error)
}
