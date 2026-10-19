package shared

import (
	"context"
	"uuid"
)

// Role is a member's role in a workspace or a project (M3 design 3.4): the
// values of Plane's ROLE_CHOICES, which the tables store.
type Role int

// The three roles. Rules compare roles by set membership, never by order, so
// any other value is allowed nothing.
const (
	RoleGuest  Role = 5
	RoleMember Role = 15
	RoleAdmin  Role = 20
)

// Action names an operation the Authorizer decides on, e.g.
// "workspace.read". Each module declares its own as constants, with an
// Actions() that lists them; the access module's rule table is keyed by
// these names (M3 design 3.4).
type Action string

// Target is what an action is on: a workspace, and for a project-level
// action a project of it; ProjectID is the zero UUID for a workspace-level
// action. CreatorID is the account that created the work item the action is
// on, or the work item a row it is on hangs from (M4 design 3.4); the zero
// UUID for every other action, and for a work item whose creator is not
// known.
type Target struct {
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	CreatorID   uuid.UUID
}

// Grant is what a decision read: the caller's active roles in the target's
// workspace and project, and whether the caller administers the project (a
// project member who is its admin or the workspace's). Use cases make the
// relative checks with it (M3 design 3.5, 3.7) instead of reading the roles
// again. OwnOnly is true for the project's guest (project role 5) when the
// project does not show its guests every work item (guest_view_all_features
// false): he sees only the work items he created, which the use cases apply
// (M4 design 3.4). It follows the project role, not the workspace's: a
// workspace admin who is only a guest in the project has it too.
type Grant struct {
	WorkspaceRole Role
	ProjectRole   Role
	ProjectAdmin  bool
	OwnOnly       bool
}

// Authorizer decides whether actor may do action on t (M3 design 3.4). It
// reads the facts on every call, in the transaction ctx carries: a write
// calls it after it has locked the parent row (M3 design 3.6 convention 2).
// A caller who cannot see the target gets ErrNotVisible, which the use case
// turns into its own resource's 404; one who can see it but whose role the
// rule does not allow gets Forbidden, with the Grant of what the decision
// read, so a use case can answer first that the guest rule hides the target
// from him (M4 design 3.4, 6.7). An action without a rule is an internal
// error: nothing is allowed.
type Authorizer interface {
	Authorize(ctx context.Context, actor Actor, action Action, t Target) (Grant, error)
}

// ErrNotVisible is the Authorizer's answer when the caller cannot see the
// target. It has no code: the use case recognizes it with errors.Is and
// answers the 404 code of what the caller named, e.g. workspace.not_found,
// which only the use case knows.
var ErrNotVisible = &Error{Kind: KindNotFound, Detail: "The target is not visible to the caller."}
