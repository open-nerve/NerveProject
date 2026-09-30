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

// InvitationPreviewer reads what an invitation's link shows.
type InvitationPreviewer interface {
	// InvitationPreview returns the undeleted invitation id of an undeleted
	// workspace as its link shows it; ErrNotFound when there is none.
	InvitationPreview(ctx context.Context, id uuid.UUID) (domain.InvitationPreview, error)
}

// InvitationLocker reads an invitation by its id, then locks it under its
// workspace's lock (M3 design 3.6 convention 2).
type InvitationLocker interface {
	// InvitationByID returns the undeleted invitation id; ErrNotFound when
	// there is none.
	InvitationByID(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
	// LockInvitation locks the undeleted invitation id FOR UPDATE until the
	// transaction ends and returns it; ErrNotFound when there is none, also
	// when it was deleted while the lock waited.
	LockInvitation(ctx context.Context, id uuid.UUID) (domain.Invitation, error)
}

// InvitationUpdater changes an invitation's role under its workspace's FOR
// SHARE.
type InvitationUpdater interface {
	InvitationLocker
	// ShareWorkspace locks the undeleted workspace id FOR SHARE until the
	// transaction ends; ErrNotFound when there is none, also when it was
	// deleted while the lock waited.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// UpdateInvitationRole sets the invitation's role, by the account by at
	// now, and returns it as stored.
	UpdateInvitationRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Invitation, error)
}

// InvitationDeleter deletes an invitation under its workspace's FOR SHARE.
type InvitationDeleter interface {
	InvitationLocker
	// ShareWorkspace is InvitationUpdater's.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// DeleteInvitation soft-deletes the invitation, by the account by at
	// now.
	DeleteInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
}

// InvitationAccepter accepts an invitation under its workspace's FOR NO KEY
// UPDATE, the lock of a write of a membership.
type InvitationAccepter interface {
	InvitationLocker
	// LockWorkspace locks the undeleted workspace id FOR NO KEY UPDATE until
	// the transaction ends; ErrNotFound when there is none, also when it was
	// deleted while the lock waited.
	LockWorkspace(ctx context.Context, id uuid.UUID) error
	// MemberOf returns userID's undeleted membership of the workspace,
	// active or ended; found is false when he has none.
	MemberOf(ctx context.Context, workspaceID, userID uuid.UUID) (m domain.Membership, found bool, err error)
	// RestoreMember makes the ended membership id active again with role,
	// by the account by at now.
	RestoreMember(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) error
	// CreateMember inserts m.
	CreateMember(ctx context.Context, m MemberRow) error
	// AcceptInvitation records the invitation as accepted by the account by
	// at now, and deletes it.
	AcceptInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
	// WorkspaceByID returns the undeleted workspace id and its number of
	// active members, without a role.
	WorkspaceByID(ctx context.Context, id uuid.UUID) (domain.Workspace, error)
}

// InvitationDecliner declines an invitation under its workspace's FOR
// SHARE.
type InvitationDecliner interface {
	InvitationLocker
	// ShareWorkspace is InvitationUpdater's.
	ShareWorkspace(ctx context.Context, id uuid.UUID) error
	// DeclineInvitation records the invitation as declined by the account
	// by at now; it stays undeleted.
	DeclineInvitation(ctx context.Context, id, by uuid.UUID, now time.Time) error
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
