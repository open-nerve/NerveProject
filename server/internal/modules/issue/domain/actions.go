package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's actions: the keys of its rows in the access module's
// rule table (M4 design 3.4).
const (
	// ActionCreate is creating a work item in a project.
	ActionCreate shared.Action = "issue.create"
	// ActionRead is reading a work item of a project.
	ActionRead shared.Action = "issue.read"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionCreate, ActionRead}
}
