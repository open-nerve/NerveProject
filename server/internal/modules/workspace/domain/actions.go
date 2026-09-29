package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The workspace module's actions: the keys of its rows in the access
// module's rule table (M3 design 3.4).
const (
	// ActionRead is reading a workspace: getWorkspace.
	ActionRead shared.Action = "workspace.read"
	// ActionUpdate is changing a workspace: updateWorkspace.
	ActionUpdate shared.Action = "workspace.update"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionRead, ActionUpdate}
}
