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

// CallerLock is identity's credential lock (M2 design 3.5), which
// identity.Provide offers (M3 design 6.5, 6.6): LockCaller locks the
// caller's account row FOR NO KEY UPDATE until the transaction ends, then
// checks under the lock that his account is active and his session or
// personal access token valid at now; 401 unauthorized otherwise. It is the
// first lock of a transaction that issues something with the caller's
// credential (M3 design 3.6 convention 1, 3.8).
type CallerLock interface {
	LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error
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

// InvitationCreator inserts a batch of invitations under the workspace's
// FOR SHARE, after it read what the batch must not repeat.
type InvitationCreator interface {
	WorkspaceSharer
	// ListMembers returns the undeleted memberships of the workspace,
	// active or not.
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error)
	// ListInvitations returns the workspace's undeleted invitations.
	ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error)
	// CreateInvitations inserts rows, one statement each, in the order
	// given, and returns them as stored, in that order; the first row whose
	// address an undeleted invitation of the workspace has is
	// *DuplicateInvitation.
	CreateInvitations(ctx context.Context, rows []InvitationRow) ([]domain.Invitation, error)
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
