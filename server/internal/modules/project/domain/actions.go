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
	// ActionMemberList is listing a project's members: listProjectMembers.
	ActionMemberList shared.Action = "project_member.list"
	// ActionMemberAdd is adding members to a project: addProjectMembers.
	ActionMemberAdd shared.Action = "project_member.add"
	// ActionJoin is joining a project: joinProject.
	ActionJoin shared.Action = "project.join"
	// ActionMemberUpdate is changing a project member's role:
	// updateProjectMember.
	ActionMemberUpdate shared.Action = "project_member.update"
	// ActionMemberRemove is removing a member from a project:
	// removeProjectMember.
	ActionMemberRemove shared.Action = "project_member.remove"
	// ActionLeave is leaving a project: leaveProject.
	ActionLeave shared.Action = "project.leave"
	// ActionStateList is listing a project's states: listStates.
	ActionStateList shared.Action = "state.list"
	// ActionStateCreate is creating a state in a project: createState.
	ActionStateCreate shared.Action = "state.create"
	// ActionStateUpdate is changing a state: updateState.
	ActionStateUpdate shared.Action = "state.update"
	// ActionStateDelete is deleting a state: deleteState.
	ActionStateDelete shared.Action = "state.delete"
	// ActionStateMarkDefault is making a state its project's default:
	// markDefaultState.
	ActionStateMarkDefault shared.Action = "state.mark_default"
	// ActionWorkspaceStateList is listing the states of a workspace's
	// projects that the caller is a member of: listWorkspaceStates.
	ActionWorkspaceStateList shared.Action = "workspace_state.list"
	// ActionLabelList is listing a project's labels: listLabels.
	ActionLabelList shared.Action = "label.list"
	// ActionLabelCreate is creating a label in a project: createLabel.
	ActionLabelCreate shared.Action = "label.create"
	// ActionLabelUpdate is changing a label: updateLabel.
	ActionLabelUpdate shared.Action = "label.update"
	// ActionLabelDelete is deleting a label and the labels under it:
	// deleteLabel.
	ActionLabelDelete shared.Action = "label.delete"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
// Each operation adds its action here with its row.
func Actions() []shared.Action {
	return []shared.Action{ActionList, ActionCreate, ActionRead, ActionCheckIdentifier, ActionUpdate, ActionArchive,
		ActionUnarchive, ActionDelete, ActionPreferencesRead, ActionPreferencesUpdate,
		ActionMemberList, ActionMemberAdd, ActionJoin, ActionMemberUpdate, ActionMemberRemove, ActionLeave,
		ActionStateList, ActionStateCreate, ActionStateUpdate, ActionStateDelete, ActionStateMarkDefault, ActionWorkspaceStateList,
		ActionLabelList, ActionLabelCreate, ActionLabelUpdate, ActionLabelDelete}
}
