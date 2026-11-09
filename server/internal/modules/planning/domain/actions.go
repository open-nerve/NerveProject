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
	// ActionModuleRead is reading a project's modules and their links.
	ActionModuleRead shared.Action = "module.read"
	// ActionModuleWrite is creating, changing, archiving and unarchiving a
	// module of a project, and adding, changing and removing its links.
	ActionModuleWrite shared.Action = "module.write"
	// ActionModuleDelete is deleting a module of a project.
	ActionModuleDelete shared.Action = "module.delete"
	// ActionCyclePreferences is reading and changing one's own display
	// settings on a cycle's page (M6 P3b spec 2.3, R6).
	ActionCyclePreferences shared.Action = "cycle.preferences"
	// ActionModulePreferences is reading and changing one's own display
	// settings on a module's page.
	ActionModulePreferences shared.Action = "module.preferences"
)

// ModuleAction reports whether a is one of the modules' actions: a write of
// it is held to the project's module toggle, any other planning write to its
// cycle toggle (M6 design 3.8). module.preferences is one, though a change
// of one's own display settings is held to no toggle (M6 P3b spec R6).
func ModuleAction(a shared.Action) bool {
	return a == ActionModuleRead || a == ActionModuleWrite || a == ActionModuleDelete || a == ActionModulePreferences
}

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionCycleRead, ActionCycleWrite, ActionCycleDelete, ActionCyclePreferences, ActionModuleRead, ActionModuleWrite,
		ActionModuleDelete, ActionModulePreferences}
}
