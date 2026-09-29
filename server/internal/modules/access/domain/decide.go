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

// Project is what the ports read about the target project.
type Project struct {
	Public bool       // network = 2
	Member Membership // the caller's membership of the project
}

// Facts are what the ports read about the caller and the target (M3 design
// 3.4). Project is nil when the target has no project that counts: a
// workspace-level target, or a project that does not exist, is deleted or
// belongs to another workspace. An archived project counts (M3 design 3.19).
type Facts struct {
	Workspace Membership
	Project   *Project
}

// Decide applies rule to f (M3 design 3.4): the Grant when the caller may
// act, shared.ErrNotVisible when the caller cannot see the target, and
// shared.Forbidden when the caller sees it but the rule does not allow the
// caller's role. A caller who is not an active member of the workspace sees
// nothing in it. Roles are compared by membership in a set, never by order:
// a role outside the three is allowed nothing and sees no project it is not
// a member of. A rule of no known level is an error: nothing is allowed.
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
	switch {
	case !ws.Active:
		return shared.Grant{}, shared.ErrNotVisible
	case !slices.Contains(rule.Roles, ws.Role):
		return shared.Grant{}, shared.Forbidden()
	}
	return shared.Grant{WorkspaceRole: ws.Role}, nil
}

// knownRoles are the three roles; any other is allowed nothing.
var knownRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}

func decideProject(rule Rule, f Facts) (shared.Grant, error) {
	ws, p := f.Workspace, f.Project
	if !ws.Active || p == nil || !sees(ws.Role, *p) {
		return shared.Grant{}, shared.ErrNotVisible
	}
	// A role outside the three, the workspace's or the project's, is allowed
	// nothing, as at the workspace level.
	if !slices.Contains(knownRoles, ws.Role) || (p.Member.Active && !slices.Contains(knownRoles, p.Member.Role)) {
		return shared.Grant{}, shared.Forbidden()
	}
	grant := shared.Grant{WorkspaceRole: ws.Role}
	if p.Member.Active {
		grant.ProjectRole = p.Member.Role
		grant.ProjectAdmin = p.Member.Role == shared.RoleAdmin || ws.Role == shared.RoleAdmin
	}
	if rule.Level == LevelVisible {
		return grant, nil
	}
	if p.Member.Active && (slices.Contains(rule.Roles, p.Member.Role) || ws.Role == shared.RoleAdmin) {
		return grant, nil
	}
	return shared.Grant{}, shared.Forbidden()
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
