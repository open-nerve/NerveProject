// Package domain holds the access module's rules (M3 design 3.4, 6.4): the
// rule table and the decision, pure functions over the facts that the ports
// read.
package domain

import (
	"maps"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Level is where a rule decides (M3 design 6.4).
type Level int

// The three levels.
const (
	// LevelWorkspace allows an active member of the target's workspace whose
	// workspace role is in the rule's roles.
	LevelWorkspace Level = iota + 1
	// LevelProject allows a caller who sees the target project and is its
	// active member, with a project role in the rule's roles or as the
	// workspace's admin.
	LevelProject
	// LevelVisible allows every caller who sees the target project: only
	// project.read and project.join.
	LevelVisible
)

// Rule is a row of the rule table: its level and the roles it allows, the
// workspace roles at LevelWorkspace and the project roles at LevelProject.
// LevelVisible has none.
type Rule struct {
	Level Level
	Roles []shared.Role
}

// rules is the rule table (M3 design 3.4): one row per action, keyed by the
// name its module declares (workspace/domain/actions.go and the like). Its
// rows are the rows of the permission matrix (M3 design 9.2); an action
// without a row is refused. Each phase adds the rows of its operations, and
// bootstrap's completeness test holds the keys equal to the modules'
// Actions().
var rules = map[shared.Action]Rule{
	"workspace.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own membership: every active member; the only admin's 409 is
	// the use case's (M3 design 3.7 rule 1, 9.2).
	"workspace.leave":       {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace_member.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The relative rules (one's own role) are the use case's (M3 design 3.4).
	"workspace_member.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Removing another member: the workspace's admins (M3 design 9.2); one's
	// own membership is the use case's 409 (3.4).
	"workspace_member.remove": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own display settings: every active member (M3 design 9.2).
	"workspace_preferences.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"workspace_preferences.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The invitations and their tokens: the workspace's admins alone (M3
	// decision 4).
	"workspace_invitation.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Only admins invite, so an invitation's role is never above its
	// inviter's (M3 design 3.8): no check of its own.
	"workspace_invitation.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace_invitation.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	"workspace_invitation.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}},
	// Every active member; each sees in the list what project.read lets
	// him read (M3 design 3.4).
	"project.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The workspace's admins and members, not its guests (M3 design 9.2;
	// Plane views/project/base.py:257).
	"project.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// Whoever sees the project (M3 design 3.4, 3.19).
	"project.read": {Level: LevelVisible},
	// As project.create (spec §3 item 13).
	"project_identifier.check": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.4: a project-level rule, which Plane's pages hold).
	"project.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As project.update (M3 design 3.4, 9.2).
	"project.archive":   {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"project.unarchive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"project.delete":    {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// Every active member of the project, his own settings (M3 design 9.2).
	"project_preferences.read":   {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"project_preferences.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// Every active member of the project (M3 design 9.2).
	"project_member.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.5, 9.2; Plane views/project/member.py:46).
	"project_member.add": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// Whoever sees the project; the use case asks the workspace's admins
	// and members of him, by set, before it looks at his membership (M3
	// design 3.5, 6.4).
	"project.join": {Level: LevelVisible},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.5, 9.2; Plane views/project/member.py:234-238); the
	// relative rules are the use case's (3.5).
	"project_member.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As project_member.update (M3 design 3.5, 9.2; Plane
	// views/project/member.py:290); one's own membership and a higher
	// role are the use case's (3.5).
	"project_member.remove": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// One's own membership: every active member of the project; the only
	// admin's 409 is the use case's (M3 design 3.7 rule 1, 9.2).
	"project.leave": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.4: not its guests, whom Plane lets change states; 9.2).
	"state.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.create.
	"state.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
}

// RuleFor returns a copy of the row of action, every field of it and its
// roles cloned, so a caller cannot change the table; ok is false when the
// table has none.
func RuleFor(action shared.Action) (Rule, bool) {
	r, ok := rules[action]
	r.Roles = slices.Clone(r.Roles)
	return r, ok
}

// RuleKeys lists the actions the table has a row for, sorted.
func RuleKeys() []shared.Action {
	return slices.Sorted(maps.Keys(rules))
}
