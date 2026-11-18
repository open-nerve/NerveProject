package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AccessFacts are what project.Provide's ProjectAccess answers the access
// module (M3 design 6.5): the project's workspace, whether it is public,
// whether the user asked about is its active member, with his project role
// then, and whether the project shows its guests every work item
// (guest_view_all_features, M4 design 3.4). bootstrap converts them into
// access's value.
type AccessFacts struct {
	WorkspaceID  uuid.UUID
	Public       bool
	Member       bool
	Role         shared.Role
	GuestViewAll bool
}

// AccessFactsMany are what project.Provide's ProjectAccessMany answers the
// access module for many accounts at once (M7 P1b spec 2.7): the facts of
// the project AccessFacts has, and the project role of each account asked
// about that is its active member, by account; one that is not is not in
// Roles. bootstrap converts them into access's value.
type AccessFactsMany struct {
	WorkspaceID  uuid.UUID
	Public       bool
	GuestViewAll bool
	Roles        map[uuid.UUID]shared.Role
}

// MemberAccessFacts are what project.Provide's MemberProjectAccess answers
// the access module of each project the user is an active member of (M7
// P1c spec 2.5): the project, whether it is public, his project role, and
// whether it shows its guests every work item, as AccessFacts has them.
// bootstrap converts them into access's value.
type MemberAccessFacts struct {
	ProjectID    uuid.UUID
	Public       bool
	Role         shared.Role
	GuestViewAll bool
}
