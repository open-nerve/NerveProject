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
// its workspace, whether it is public, whether the user asked about is its
// active member, with his project role then, and whether it shows its
// guests every work item (guest_view_all_features, M4 design 3.4).
type ProjectFacts struct {
	WorkspaceID  uuid.UUID
	Public       bool
	Member       bool
	Role         shared.Role
	GuestViewAll bool
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

// WorkspaceRolesMany reads many accounts' memberships of a workspace at once,
// for DecideMany (M7 P1b spec 2.7). The workspace module implements it
// (workspace.Provide); bootstrap wires it.
type WorkspaceRolesMany interface {
	// ActiveRoles returns the role of each of accountIDs whose membership of
	// workspaceID is active, as ActiveRole reads one's: its row is_active
	// and not deleted, and the workspace not deleted. An account without one
	// is not in it. It reads in the transaction ctx carries.
	ActiveRoles(ctx context.Context, workspaceID uuid.UUID, accountIDs []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}

// ProjectFactsMany are what ProjectAccessMany reads of a project for many
// accounts' decisions: the facts of the project ProjectFacts has, and the
// project role of each account asked about that is its active member, by
// account; one that is not is not in Roles.
type ProjectFactsMany struct {
	WorkspaceID  uuid.UUID
	Public       bool
	GuestViewAll bool
	Roles        map[uuid.UUID]shared.Role
}

// ProjectAccessMany reads a project for many accounts' decisions at once (M7
// P1b spec 2.7). The project module implements it (project.Provide);
// bootstrap wires it, converting the facts.
type ProjectAccessMany interface {
	// ProjectFactsMany returns the facts of the undeleted project projectID,
	// archived or not, for accountIDs; found is false for a project that
	// does not exist or is deleted. It reads in the transaction ctx carries,
	// as ProjectFacts does.
	ProjectFactsMany(ctx context.Context, projectID uuid.UUID, accountIDs []uuid.UUID) (f ProjectFactsMany, found bool, err error)
}
