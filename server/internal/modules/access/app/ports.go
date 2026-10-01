// Package app holds the access module's Authorizer (M3 design 6.4): it reads
// the facts through ports, each implemented by the module that owns the
// table, and decides with the rule table of domain.
package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// WorkspaceRoles reads a caller's membership of a workspace (M3 design 6.5).
// The workspace module implements it (workspace.Provide); bootstrap wires it.
type WorkspaceRoles interface {
	// ActiveRole returns userID's role in workspaceID when the membership is
	// active: its row is_active and not deleted, and the workspace not
	// deleted. ok is false otherwise. It reads in the transaction ctx
	// carries, so a write that locked the workspace row reads the role
	// committed before its lock (M3 design 3.6 convention 2).
	ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (role shared.Role, ok bool, err error)
}

// ProjectFacts are what ProjectAccess reads of a project for a decision:
// its workspace, whether it is public, and whether the user asked about is
// its active member, with his project role then.
type ProjectFacts struct {
	WorkspaceID uuid.UUID
	Public      bool
	Member      bool
	Role        shared.Role
}

// ProjectAccess reads a project for a decision (M3 design 6.5). The project
// module implements it (project.Provide); bootstrap wires it, converting the
// facts.
type ProjectAccess interface {
	// ProjectFacts returns the facts of the undeleted project projectID,
	// archived or not, for userID; found is false for a project that does
	// not exist or is deleted. It reads in the transaction ctx carries, as
	// ActiveRole does.
	ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (f ProjectFacts, found bool, err error)
}
