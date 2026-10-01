// Package domain holds the project module's rules (M3 design 6.3): pure
// functions and values.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's actions: the keys of its rows in the access module's
// rule table (M3 design 3.4).
const (
	// ActionList is listing a workspace's projects: listProjects.
	ActionList shared.Action = "project.list"
	// ActionCreate is creating a project in a workspace: createProject.
	ActionCreate shared.Action = "project.create"
	// ActionRead is reading a project: getProject.
	ActionRead shared.Action = "project.read"
	// ActionCheckIdentifier is asking whether an identifier is available
	// in a workspace: checkProjectIdentifier.
	ActionCheckIdentifier shared.Action = "project_identifier.check"
	// ActionUpdate is changing a project: updateProject.
	ActionUpdate shared.Action = "project.update"
	// ActionArchive is archiving a project: archiveProject.
	ActionArchive shared.Action = "project.archive"
	// ActionUnarchive is unarchiving a project: unarchiveProject.
	ActionUnarchive shared.Action = "project.unarchive"
	// ActionDelete is deleting a project: deleteProject.
	ActionDelete shared.Action = "project.delete"
	// ActionPreferencesRead is reading one's display settings in a project:
	// getProjectPreferences.
	ActionPreferencesRead shared.Action = "project_preferences.read"
	// ActionPreferencesUpdate is changing them: updateProjectPreferences.
	ActionPreferencesUpdate shared.Action = "project_preferences.update"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
// Each operation adds its action here with its row.
func Actions() []shared.Action {
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier, ActionUpdate, ActionArchive,
		ActionUnarchive, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate}
}
