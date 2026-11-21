package app

import "uuid"

// NavigationTarget is what planning.Provide hands the navigation module
// about a cycle or a module a favorite names (M7 P3c spec 2.4, 2.5): its
// workspace and its project, its name, which the favorite is answered
// with, and whether it is archived, which hides it. bootstrap converts it
// into navigation's value.
type NavigationTarget struct {
	WorkspaceID, ProjectID uuid.UUID
	Name                   string
	Archived               bool
}
