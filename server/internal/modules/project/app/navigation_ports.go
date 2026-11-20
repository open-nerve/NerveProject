package app

import "uuid"

// NavigationFacts are what project.Provide hands the navigation module about
// a project (M7 P3b spec 2.1): its workspace, and whether its views are
// turned on (issue_views_view). bootstrap converts them into navigation's
// value.
type NavigationFacts struct {
	WorkspaceID    uuid.UUID
	IssueViewsView bool
}
