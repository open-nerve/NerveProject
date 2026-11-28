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
	// workspace's admin; and, by a rule with AllowCreator, the creator of the
	// target work item when his project role is member or admin.
	LevelProject
	// LevelVisible allows every caller who sees the target project: only
	// project.read and project.join.
	LevelVisible
)

// Rule is a row of the rule table: its level and the roles it allows, the
// workspace roles at LevelWorkspace and the project roles at LevelProject.
// LevelVisible has none. AllowCreator lets a project-level rule allow the
// creator of the target work item besides those roles, while he is an
// active member of the project of the role member or above (M4 design 3.4:
// a guest who created it is not allowed; Plane's creator=True looks at the
// workspace membership alone).
type Rule struct {
	Level        Level
	Roles        []shared.Role
	AllowCreator bool
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
	// Every active member; the list holds the states of the projects he is
	// an active member of, those state.list lets him list (M3 design 3.4).
	"workspace_state.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// Every active member; the list holds the labels of the unarchived
	// projects he is an active member of, which label.list lets him list
	// (M7/P6b spec 2.1).
	"workspace_label.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
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
	// Every active member of the project (M3 design 9.2).
	"state.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// The project's admins, and its members who are the workspace's admins
	// (M3 design 3.4: not its guests, whom Plane lets change states; 9.2).
	"state.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.create.
	"state.update":       {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"state.delete":       {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"state.mark_default": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As state.list (M3 design 9.2).
	"label.list": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As state.create (M3 design 3.4, 9.2).
	"label.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"label.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	"label.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// The project's admins and members, not its guests (M4 design 3.4).
	"issue.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// The project's admins, members and guests; a guest who sees only his
	// own work items finds no other, the issue module's guest rule (M4
	// design 3.4).
	"issue_comment.write":  {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"issue_reaction.write": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"issue_comment.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"issue.read":           {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As issue.create (M4 design 3.4).
	"issue_description_version.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"issue.update":                   {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// The project's admins, and the work item's creator while he is a member
	// or an admin of the project, not a guest who created it (M4 design 3.4).
	"issue.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true},
	// As issue.create: archiving and unarchiving a work item (M4 design 3.4,
	// 3.8).
	"issue.archive": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// As issue.read: every active member of the project submits to its
	// intake, its guest too (M7 design 3.10); whether the intake is turned
	// on is the issue module's, under the project's lock (P2a spec 2.4).
	"intake.submit": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As state.create: the project's admins, and the workspace's admin who
	// is its active member, triage its intake (M7 design 3.10, 15 B).
	"intake.triage": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}},
	// As issue.read: every active member of the project reads its intake,
	// its guest too, who sees only his own submissions when he sees only
	// his own work items: the issue module's guest rule; turned off or not
	// (M7 design 3.10, 3.11, P2b spec 2.3).
	"intake.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As intake.submit: every active member of the project changes a work
	// item of its intake, and writes its assets while it is in the intake;
	// that one who is not its admin submitted it, and that a guest changes
	// only its name and description of its content, are the issue module's
	// relative rules (M7 design 3.6, 3.10, P2b spec 2.4, P2c spec 2.2).
	"intake.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As issue.delete: the project's admins, and the work item's submitter,
	// its creator, while a member or an admin of the project, not a guest
	// who submitted it (M7 design 3.10, P2b spec 2.5).
	"intake.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true},
	// Every active member of the workspace: getIssueByIdentifier's decision
	// in the workspace, before issue.read on the project of the work item
	// it finds, which the guest rule follows (M4 design 3.4, 3.7).
	"issue_draft.write": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"member_issue.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"issue.search":      {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As issue.search: every active member of the workspace counts the
	// pending work items of the intakes of the projects he sees; which
	// projects, and whose work items a guest counts, are the issue
	// module's (M7 design 3.10, P2b spec 2.6).
	"intake.count": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As issue.search: every active member of the workspace lists the work
	// items of the projects he sees in it; which projects, and whose work
	// items a guest sees there, are the issue module's (M7 design 3.10, P3a
	// spec 2.3). No role of the three is refused, so the list has no 403 (R7).
	"workspace_issue.list": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// One's own notifications: every active member of the workspace (M7
	// design 3.10). Which of them he sees is the notification module's, by
	// what he reads now (M7 P1c spec 2.5).
	"notification.read":  {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"notification.write": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// Every active member of the project reads cycles; the guest's OwnOnly
	// filtering is the planning module's (M6 design 3.7).
	"cycle.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As issue.create: creating, changing, archiving and unarchiving a
	// cycle; the planning module refuses an OwnOnly grant (M6 design 3.7).
	"cycle.write": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// As issue.delete: the project's admins, and the cycle's creator while a
	// member or an admin of the project (M6 design 3.7).
	"cycle.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true},
	// As cycle.read: every active member of the project reads and changes
	// his own display settings on a cycle's page; the planning module
	// refuses no OwnOnly grant there, nor an archived project or a toggle
	// turned off (M6 P3b spec 2.3, R6).
	"cycle.preferences": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As cycle.read: every active member of the project reads modules and
	// their links; the guest's OwnOnly filtering is the planning module's
	// (M6 design 3.7).
	"module.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As cycle.write: creating, changing, archiving and unarchiving a
	// module, and writing its links; the planning module refuses an OwnOnly
	// grant (M6 design 3.7).
	"module.write": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// As cycle.delete: the project's admins, and the module's creator while
	// a member or an admin of the project (M6 design 3.7).
	"module.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true},
	// As cycle.preferences, on a module's page.
	"module.preferences": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As cycle.read: every active member of the project reads its saved
	// views; whose views an OwnOnly grant sees is the navigation module's
	// (M7 design 3.7, 3.10, P3b spec 2.2).
	"view.read": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	// As cycle.write: the project's admins and members create, change and
	// delete its saved views; the toggle issue_views_view, that only the
	// owner changes a view and that the owner or the project's admin
	// deletes one are the navigation module's (P3b spec 2.2).
	"view.create": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"view.update": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"view.delete": {Level: LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// A workspace's saved views: its admins and members, not its guests,
	// who have the four default views only (M7 design 3.7, 3.10, P3b spec
	// 2.2, R1). That only the owner changes a view and that the owner or
	// the workspace's admin deletes one are the navigation module's.
	"workspace_view.read":   {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"workspace_view.create": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"workspace_view.update": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"workspace_view.delete": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// One's own favorites: the workspace's admins and members, not its
	// guests (M7 design 3.8, 3.10, P3c spec 2.2; Plane's favorites API
	// alike). That one reads and writes his own only, and which targets he
	// sees and may favor, are the navigation module's.
	"favorite.read":  {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	"favorite.write": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember}},
	// One's own recent visits: every active member of the workspace, its
	// guests too (M7 design 3.9, 3.10, P3d spec 2.2; Plane's list alike).
	// That one reads and records his own only, and that a target is one he
	// sees now, are the navigation module's.
	"recent_visit.read":  {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
	"recent_visit.write": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
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
