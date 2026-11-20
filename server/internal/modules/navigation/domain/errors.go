package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The navigation module's errors (P3b spec 2.4, 2.5).
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
)
