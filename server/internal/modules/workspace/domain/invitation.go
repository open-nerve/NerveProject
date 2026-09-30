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

// InvitationWithToken is an invitation and the token of its link, which
// only who may manage the workspace's invitations is given (M3 design 5.2).
type InvitationWithToken struct {
	Invitation
	Token string
}
