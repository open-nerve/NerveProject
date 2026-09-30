package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The workspace module's actions: the keys of its rows in the access
// module's rule table (M3 design 3.4).
const (
	// ActionRead is reading a workspace: getWorkspace.
	ActionRead shared.Action = "workspace.read"
	// ActionUpdate is changing a workspace: updateWorkspace.
	ActionUpdate shared.Action = "workspace.update"
	// ActionDelete is deleting a workspace: deleteWorkspace.
	ActionDelete shared.Action = "workspace.delete"
	// ActionMemberList is listing a workspace's members:
	// listWorkspaceMembers.
	ActionMemberList shared.Action = "workspace_member.list"
	// ActionMemberUpdate is changing a member's role: updateWorkspaceMember.
	ActionMemberUpdate shared.Action = "workspace_member.update"
	// ActionPreferencesRead is reading one's display settings in a
	// workspace: getWorkspacePreferences.
	ActionPreferencesRead shared.Action = "workspace_preferences.read"
	// ActionPreferencesUpdate is changing them: updateWorkspacePreferences.
	ActionPreferencesUpdate shared.Action = "workspace_preferences.update"
	// ActionInvitationList is listing a workspace's invitations, with their
	// tokens: listWorkspaceInvitations.
	ActionInvitationList shared.Action = "workspace_invitation.list"
	// ActionInvitationCreate is inviting addresses to a workspace:
	// createWorkspaceInvitations.
	ActionInvitationCreate shared.Action = "workspace_invitation.create"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionRead, ActionUpdate, ActionDelete, ActionMemberList, ActionMemberUpdate, ActionPreferencesRead, ActionPreferencesUpdate,
		ActionInvitationList, ActionInvitationCreate}
}
