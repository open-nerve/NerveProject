package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The navigation module's errors (P3b spec 2.4, 2.5; P3c spec 2.4).
var (
	// ErrViewNotFound answers, all with this one value, every view that
	// /views/{view_id} cannot answer (spec 2.5): one that does not exist,
	// a deleted one, one whose project or workspace the caller does not
	// see or is deleted, and one the guest rule hides from him
	// (View.HiddenFrom). Its answer never tells which.
	ErrViewNotFound = shared.NewError(shared.KindNotFound, "navigation.view_not_found", "The view does not exist, or you cannot see it.")
	// ErrViewsDisabled answers a write on the views of a project whose
	// views are turned off (issue_views_view, M7 design 3.7); they are
	// read as ever.
	ErrViewsDisabled = shared.NewError(shared.KindConflict, "navigation.views_disabled", "Views are turned off in this project; turn them on to change them.")
	// ErrProjectNotFound answers a call on a project's views, by the
	// project's id, when the project is not there, is deleted, or the
	// caller does not see it (spec 2.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrWorkspaceNotFound answers a call on a workspace's views, by the
	// workspace's slug, when the slug names no undeleted workspace or the
	// caller is not its active member (spec 2.2).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
	// ErrFavoriteNotFound answers, all with this one value, every favorite
	// that /favorites/{favorite_id} cannot answer (P3c spec 2.7, R7): one
	// that does not exist, a deleted one, another's, one of a workspace
	// the caller no longer sees, and one whose target he no longer sees.
	// Its answer never tells which.
	ErrFavoriteNotFound = shared.NewError(shared.KindNotFound, "navigation.favorite_not_found", "The favorite does not exist, or you cannot see it.")
	// ErrFavoriteFolderNameTaken answers a folder's creation or renaming
	// to a name, in any case, that another of the caller's undeleted
	// folders in the workspace has (P3c spec 2.5, D41); renaming a folder
	// to its own name, in another case, is not one.
	ErrFavoriteFolderNameTaken = shared.NewError(shared.KindConflict, "navigation.favorite_folder_name_taken",
		"You have a folder of this name in this workspace already.")
)
