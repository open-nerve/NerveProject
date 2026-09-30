package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A write on an invitation named by its id (M3 design 3.6 convention 2)
// reads the invitation for its workspace (readInvitation), locks the
// workspace, then locks the invitation row FOR UPDATE and reads it again
// (lockInvitation): its decisions and checks see what committed before, and
// nothing changes the row until the write ends. A response first locks the
// caller's account row (3.6 conventions 1 and 6), then reads and locks as
// above.

// readInvitation reads the undeleted invitation id, without a lock;
// domain.ErrInvitationNotFound when there is none.
func readInvitation(ctx context.Context, invitations InvitationLocker, id uuid.UUID) (domain.Invitation, error) {
	inv, err := invitations.InvitationByID(ctx, id)
	if err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	return inv, nil
}

// lockInvitation locks inv's workspace with lockWorkspace, then inv's row
// FOR UPDATE, and returns the row read under the lock. A workspace deleted,
// or an invitation deleted (by an acceptance too) while the locks waited,
// is domain.ErrInvitationNotFound.
func lockInvitation(ctx context.Context, invitations InvitationLocker, lockWorkspace func(ctx context.Context, id uuid.UUID) error,
	inv domain.Invitation) (domain.Invitation, error) {
	if err := lockWorkspace(ctx, inv.WorkspaceID); err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	locked, err := invitations.LockInvitation(ctx, inv.ID)
	if err != nil {
		return domain.Invitation{}, invitationNotFound(err)
	}
	return locked, nil
}

// invitationNotFound turns ErrNotFound into domain.ErrInvitationNotFound.
func invitationNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrInvitationNotFound
	}
	return err
}
