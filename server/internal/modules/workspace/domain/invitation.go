package domain

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Invitation is an undeleted row of workspace_member_invites (M3 design
// 4.4, 5.2): an invitation of an address to a workspace, with a role,
// pending or declined. An accepted invitation is deleted as it is accepted,
// so the stores never answer one.
type Invitation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string // normalized (M3 design 3.13)
	Role        shared.Role
	Accepted    bool
	RespondedAt *time.Time
	CreatedAt   time.Time
	CreatedByID *uuid.UUID
}

// Responded reports whether the invitation has been answered: declined,
// for an undeleted one (M3 design 3.8).
func (i Invitation) Responded() bool {
	return i.RespondedAt != nil
}
