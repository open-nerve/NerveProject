package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The ports of the invitations' use cases (M3 design 3.8, 6.5).

// InvitationRow is an invitation to insert: a checked, normalized address
// and a role, its id, its inviter and the time of the use case's clock.
type InvitationRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// DuplicateInvitation is a store's answer to an insert that the unique key
// of (workspace, address) refused: an undeleted invitation of the
// workspace, pending or declined, has Email (M3 design 3.8).
type DuplicateInvitation struct {
	Email string
}

func (e *DuplicateInvitation) Error() string {
	return "an undeleted invitation of the workspace has the address"
}
