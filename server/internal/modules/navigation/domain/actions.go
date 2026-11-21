// Package domain holds the navigation module's rules (M7 design 3.7, 3.10,
// P3b spec 2): its actions, the keys of its rows in the access module's
// rule table; a saved view, its content, a change of it and the checks of
// both; who sees, changes and deletes a view; and the module's errors.
package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The navigation module's actions (P3b spec 2.2, P3c spec 2.2). A
// project's saved views are the project's: its admins, members and guests
// read them, its admins and members create, change and delete them. A
// workspace's saved views are the workspace's, for its admins and members
// alone. That only the owner changes a view, that the owner or the admin
// deletes one, whose views an OwnOnly grant sees and the project's toggle
// issue_views_view are the module's own rules. One's favorites in a
// workspace are for its admins and members, a workspace-level read and
// write; that one reads and writes his own only and which targets he sees
// and may favor are the module's.
const (
	ActionViewRead            shared.Action = "view.read"
	ActionViewCreate          shared.Action = "view.create"
	ActionViewUpdate          shared.Action = "view.update"
	ActionViewDelete          shared.Action = "view.delete"
	ActionWorkspaceViewRead   shared.Action = "workspace_view.read"
	ActionWorkspaceViewCreate shared.Action = "workspace_view.create"
	ActionWorkspaceViewUpdate shared.Action = "workspace_view.update"
	ActionWorkspaceViewDelete shared.Action = "workspace_view.delete"
	ActionFavoriteRead        shared.Action = "favorite.read"
	ActionFavoriteWrite       shared.Action = "favorite.write"
)

// Actions lists the module's actions. bootstrap's test holds the union of
// every module's Actions equal to the rule table's keys (M3 design 3.4).
func Actions() []shared.Action {
	return []shared.Action{ActionViewRead, ActionViewCreate, ActionViewUpdate, ActionViewDelete,
		ActionWorkspaceViewRead, ActionWorkspaceViewCreate, ActionWorkspaceViewUpdate, ActionWorkspaceViewDelete,
		ActionFavoriteRead, ActionFavoriteWrite}
}

// Verb is what an operation does with a view: read, create, update or
// delete it.
type Verb int

// The four verbs.
const (
	Read Verb = iota + 1
	Create
	Update
	Delete
)

// actionsOf are each kind's actions, by verb.
var actionsOf = map[Kind]map[Verb]shared.Action{
	ProjectView:   {Read: ActionViewRead, Create: ActionViewCreate, Update: ActionViewUpdate, Delete: ActionViewDelete},
	WorkspaceView: {Read: ActionWorkspaceViewRead, Create: ActionWorkspaceViewCreate, Update: ActionWorkspaceViewUpdate, Delete: ActionWorkspaceViewDelete},
}

// ActionOf is the action of verb on a view of kind k (spec 2.2, 2.5): a
// project's view's view.*, which Authorize decides on the view's project;
// a workspace's view's workspace_view.*, decided on its workspace. A
// /views/{view_id} call reads the view first and chooses by its kind.
func ActionOf(k Kind, verb Verb) shared.Action { return actionsOf[k][verb] }
