package app

import (
	"context"
	"errors"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Reactivation is what `nerve workspaces reactivate-member` found and did.
type Reactivation struct {
	Email string      // the account's address, normalized
	Role  shared.Role // the membership's role, kept
	// AlreadyActive is set when the membership was active: nothing changed.
	AlreadyActive bool
	// EndedProjectMemberships is the number of the member's ended
	// memberships of the workspace's projects, which stay ended: each comes
	// back when he joins its project, which a workspace admin may do for
	// any project and a member for a public one, or when a project admin
	// adds him again (M3 design 3.5, 3.11).
	EndedProjectMemberships int
	// AccountActive is false for a deactivated account, which the command
	// reactivates all the same: `nerve users activate` is the next step.
	AccountActive bool
}

// ReactivateMember restores an ended membership, for the server's
// administrator: `nerve workspaces reactivate-member` (M3 design 3.11, as
// Plane's reactivate_workspace_member).
type ReactivateMember struct {
	accounts Accounts
	members  MemberReactivator
	counts   ProjectMembershipCounts
	tx       shared.TxManager
	clock    Clock
	logger   *slog.Logger
}

// NewReactivateMember returns the use case.
func NewReactivateMember(accounts Accounts, members MemberReactivator, counts ProjectMembershipCounts, tx shared.TxManager, clock Clock,
	logger *slog.Logger) *ReactivateMember {
	return &ReactivateMember{accounts: accounts, members: members, counts: counts, tx: tx, clock: clock, logger: logger}
}

// Execute makes the membership of the account of email in the workspace
// slug active again, its role kept, in one transaction (M3 design 3.6's
// lock table): the account row FOR SHARE first, its state read under the
// lock, a deactivated account allowed (3.6 convention 6, its one
// exception); then the workspace FOR NO KEY UPDATE, the membership read
// under it, the clock (3.3), the reactivation, and the count of his ended
// project memberships, read without a lock. The address is normalized as
// registration does it. No such account is workspace.account_not_found, no
// such workspace workspace.slug_not_found, no membership of it
// workspace.never_a_member; an active one is reported and changes nothing.
// A reactivation is logged once.
func (u *ReactivateMember) Execute(ctx context.Context, slug, email string) (Reactivation, error) {
	r := Reactivation{Email: shared.NormalizeEmail(email)}
	var workspaceID, userID uuid.UUID
	err := u.tx.WithinTx(ctx, func(ctx context.Context) error {
		state, found, err := u.accounts.ShareAccountByEmail(ctx, r.Email)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrAccountNotFound
		}
		r.AccountActive, userID = state.Active, state.ID
		workspaceID, err = u.members.LockWorkspaceBySlug(ctx, slug)
		switch {
		case errors.Is(err, ErrNotFound):
			return domain.ErrSlugNotFound
		case err != nil:
			return err
		}
		m, found, err := u.members.MemberOf(ctx, workspaceID, userID)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrNeverAMember
		}
		r.Role, r.AlreadyActive = m.Role, m.IsActive
		if m.IsActive {
			return nil
		}
		if err := u.members.ReactivateMember(ctx, workspaceID, userID, u.clock.Now()); err != nil {
			return err
		}
		r.EndedProjectMemberships, err = u.counts.CountInactive(ctx, workspaceID, userID)
		return err
	})
	if err != nil {
		return Reactivation{}, err
	}
	if !r.AlreadyActive {
		u.logger.InfoContext(ctx, "workspace member reactivated", slog.String("workspace_id", workspaceID.String()),
			slog.String("user_id", userID.String()), slog.String("by", byCLI))
	}
	return r, nil
}
