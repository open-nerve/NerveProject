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
