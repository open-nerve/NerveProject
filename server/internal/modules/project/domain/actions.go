// Package domain holds the project module's rules (M3 design 6.3): pure
// functions and values.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's actions: the keys of its rows in the access module's
// rule table (M3 design 3.4).
const (
	// ActionCreate is creating a project in a workspace: createProject.
	ActionCreate shared.Action = "project.create"
	// ActionRead is reading a project: getProject.
	ActionRead shared.Action = "project.read"
	// ActionCheckIdentifier is asking whether an identifier is available
	// in a workspace: checkProjectIdentifier.
	ActionCheckIdentifier shared.Action = "project_identifier.check"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
// Each operation adds its action here with its row.
func Actions() []shared.Action {
	return []shared.Action{ActionCreate, ActionRead, ActionCheckIdentifier}
}
