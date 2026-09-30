package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The ports of the invitations' use cases (M3 design 3.8, 6.5).

// InvitationMAC tags the messages of the invitations' tokens: identity's MAC
// of the purpose "workspace-invitation", whose key is derived from the
// signing key (M3 design 3.8, 6.5, 11.1). Verify compares in constant time.
type InvitationMAC interface {
	Tag(message []byte) [16]byte
	Verify(message []byte, tag [16]byte) bool
}

// InvitationLister reads a workspace and lists its invitations.
type InvitationLister interface {
	WorkspaceFinder
	// ListInvitations returns the workspace's undeleted invitations,
	// pending or declined, newest first, then by id (M3 design 3.12).
	ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error)
}

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
