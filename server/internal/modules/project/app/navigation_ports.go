package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NavigationFacts are what project.Provide hands the navigation module about
// a project (M7 P3b spec 2.1): its workspace, and whether its views are
// turned on (issue_views_view). bootstrap converts them into navigation's
// value.
type NavigationFacts struct {
	WorkspaceID    uuid.UUID
	IssueViewsView bool
}

// NavigationTarget is what project.Provide hands the navigation module about
// a project a favorite or a recent visit names (M7 P3c spec 2.4, 2.5, P3d
// spec 2.4, R4): its workspace; its name and its icon, which the favorite
// is answered with, and its identifier too, which a recent visit is; and
// whether it is archived, which hides it from the favorites alone.
// bootstrap converts it into navigation's value, whose fields are these.
type NavigationTarget struct {
	WorkspaceID uuid.UUID
	Name        string
	Identifier  string
	LogoProps   shared.LogoProps
	Archived    bool
}
