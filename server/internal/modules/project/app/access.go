package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AccessFacts are what project.Provide's ProjectAccess answers the access
// module (M3 design 6.5): the project's workspace, whether it is public, and
// whether the user asked about is its active member, with his project role
// then. bootstrap converts them into access's value.
type AccessFacts struct {
	WorkspaceID uuid.UUID
	Public      bool
	Member      bool
	Role        shared.Role
}
