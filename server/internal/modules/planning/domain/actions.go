package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The planning module's actions: the keys of its rows in the access module's
// rule table (M6 design 3.7).
const (
	// ActionCycleRead is reading a project's cycles.
	ActionCycleRead shared.Action = "cycle.read"
	// ActionCycleWrite is creating, changing, archiving and unarchiving a
	// cycle of a project.
	ActionCycleWrite shared.Action = "cycle.write"
	// ActionCycleDelete is deleting a cycle of a project.
	ActionCycleDelete shared.Action = "cycle.delete"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionCycleRead, ActionCycleWrite, ActionCycleDelete}
}
