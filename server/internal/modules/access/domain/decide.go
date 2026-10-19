package domain

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Membership is the caller's membership of a workspace or a project as a
// port read it. Active is false when there is none that counts: no row,
// is_active false, the row deleted, or the workspace or project deleted.
type Membership struct {
	Active bool
	Role   shared.Role
}

// Project is what the ports read about the target project: whether it is
// public, the caller's membership of it, and whether it shows its guests
// every work item (guest_view_all_features, M4 design 3.4).
type Project struct {
	Public       bool // network = 2
	Member       Membership
	GuestViewAll bool
}

// Facts are what the ports read about the caller and the target (M3 design
// 3.4). Project is nil when the target has no project that counts: a
// workspace-level target, or a project that does not exist, is deleted or
// belongs to another workspace. An archived project counts (M3 design 3.19).
// Creator is true when the caller created the work item the target names
// (shared.Target.CreatorID, M4 design 3.4).
type Facts struct {
	Workspace Membership
	Project   *Project
	Creator   bool
}

// Decide applies rule to f (M3 design 3.4): the Grant when the caller may
// act, shared.ErrNotVisible when the caller cannot see the target, and
// shared.Forbidden when the caller sees it but the rule does not allow the
// caller's role, with the Grant of what it read, so that a use case can
// answer first that the guest rule hides the target from him (M4 design 3.4).
// A caller who is not an active member of the workspace sees nothing in it.
// Roles are compared by membership in a set, never by order: a role outside
// the three is allowed nothing and sees no project it is not a member of. A
// rule of no known level is an error: nothing is allowed.
func Decide(rule Rule, f Facts) (shared.Grant, error) {
	switch rule.Level {
	case LevelWorkspace:
		return decideWorkspace(rule, f.Workspace)
	case LevelProject, LevelVisible:
		return decideProject(rule, f)
	}
	return shared.Grant{}, fmt.Errorf("access: a rule of unknown level %d", rule.Level)
}

func decideWorkspace(rule Rule, ws Membership) (shared.Grant, error) {
	if !ws.Active {
		return shared.Grant{}, shared.ErrNotVisible
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if !slices.Contains(rule.Roles, ws.Role) {
		return grant, shared.Forbidden()
	}
	return grant, nil
}

// knownRoles are the three roles; any other is allowed nothing.
var knownRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}

// creatorRoles are the project roles of a creator AllowCreator allows: an
// active member of the project, of the role member or above, by set (M4
// design 3.4).
var creatorRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember}

func decideProject(rule Rule, f Facts) (shared.Grant, error) {
	ws, p := f.Workspace, f.Project
	if !ws.Active || p == nil || !sees(ws.Role, *p) {
		return shared.Grant{}, shared.ErrNotVisible
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if p.Member.Active {
		grant.ProjectRole = p.Member.Role
		grant.ProjectAdmin = p.Member.Role == shared.RoleAdmin || ws.Role == shared.RoleAdmin
		// By the project role alone: the workspace's admin who is only a
		// guest in the project sees only his own work items too (M4 design
		// 3.4).
		grant.OwnOnly = p.Member.Role == shared.RoleGuest && !p.GuestViewAll
	}
	// A role outside the three, the workspace's or the project's, is allowed
	// nothing, as at the workspace level.
	if !slices.Contains(knownRoles, ws.Role) || (p.Member.Active && !slices.Contains(knownRoles, p.Member.Role)) {
		return grant, shared.Forbidden()
	}
	if rule.Level == LevelVisible {
		return grant, nil
	}
	if p.Member.Active && (slices.Contains(rule.Roles, p.Member.Role) || ws.Role == shared.RoleAdmin ||
		rule.AllowCreator && f.Creator && slices.Contains(creatorRoles, p.Member.Role)) {
		return grant, nil
	}
	return grant, shared.Forbidden()
}

// sees reports whether an active workspace member with role wsRole sees p:
// the workspace's admin sees every project, an active project member sees
// the project, and a workspace member or admin sees a public project. A
// guest sees only the projects he is a member of, public or not (Plane
// views/project/base.py:198-222).
func sees(wsRole shared.Role, p Project) bool {
	return wsRole == shared.RoleAdmin || p.Member.Active ||
		(p.Public && slices.Contains([]shared.Role{shared.RoleMember, shared.RoleAdmin}, wsRole))
}
