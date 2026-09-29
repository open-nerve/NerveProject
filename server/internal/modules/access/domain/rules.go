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
	"workspace.read": {Level: LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}},
}

// RuleFor returns a copy of the row of action, so a caller cannot change
// the table; ok is false when the table has none.
func RuleFor(action shared.Action) (Rule, bool) {
	r, ok := rules[action]
	return Rule{Level: r.Level, Roles: slices.Clone(r.Roles)}, ok
}

// RuleKeys lists the actions the table has a row for, sorted.
func RuleKeys() []shared.Action {
	return slices.Sorted(maps.Keys(rules))
}
