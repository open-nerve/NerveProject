package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The deactivation's statements (app.AllMembershipsEnder) over the
// memberships fakeWorkspaces holds: each logs its call and fails with the
// error endErrs sets for it, wrapped as the store wraps it.

// LockMemberWorkspaces answers the workspaces of which userID has an active
// membership, in id order.
func (f *fakeWorkspaces) LockMemberWorkspaces(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockMemberWorkspaces %s", userID)
	if err := f.endErrs["LockMemberWorkspaces"]; err != nil {
		return nil, fmt.Errorf("lock the member's workspaces: %w", err)
	}
	var ids []uuid.UUID
	for workspace, list := range f.memberships {
		if slices.ContainsFunc(list, func(m domain.Membership) bool { return m.MemberID == userID && m.IsActive }) {
			ids = append(ids, workspace)
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return ids, nil
}

// SoleAdmin answers whether userID is the only active admin of one of the
// workspaces that has another active member.
func (f *fakeWorkspaces) SoleAdmin(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	f.log.add(ctx, "SoleAdmin %v %s", workspaceIDs, userID)
	if err := f.endErrs["SoleAdmin"]; err != nil {
		return false, fmt.Errorf("look for a workspace he is the only admin of: %w", err)
	}
	return slices.ContainsFunc(workspaceIDs, func(w uuid.UUID) bool {
		list := f.memberships[w]
		admin := func(m domain.Membership) bool { return m.Role == shared.RoleAdmin && m.IsActive }
		other := func(m domain.Membership) bool { return m.MemberID != userID && m.IsActive }
		return slices.ContainsFunc(list, func(m domain.Membership) bool { return m.MemberID == userID && admin(m) }) &&
			!slices.ContainsFunc(list, func(m domain.Membership) bool { return other(m) && admin(m) }) && slices.ContainsFunc(list, other)
	}), nil
}

func (f *fakeWorkspaces) DeleteInvitationsTo(ctx context.Context, email string, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteInvitationsTo %s by %s at %s", email, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["DeleteInvitationsTo"]; err != nil {
		return fmt.Errorf("delete the invitations to the address: %w", err)
	}
	return nil
}

func (f *fakeWorkspaces) DeleteInvitationsOfWorkspacesLeftEmpty(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID,
	now time.Time) error {
	f.log.add(ctx, "DeleteInvitationsOfWorkspacesLeftEmpty %v %s by %s at %s", workspaceIDs, userID, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["DeleteInvitationsOfWorkspacesLeftEmpty"]; err != nil {
		return fmt.Errorf("delete the invitations of the workspaces left empty: %w", err)
	}
	return nil
}

// EndWorkspaceMemberships also ends the memberships the fake holds, as the
// store ends the rows.
func (f *fakeWorkspaces) EndWorkspaceMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndWorkspaceMemberships %v %s by %s at %s", workspaceIDs, userID, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["EndWorkspaceMemberships"]; err != nil {
		return fmt.Errorf("end the workspace memberships: %w", err)
	}
	for _, w := range workspaceIDs {
		for i, m := range f.memberships[w] {
			if m.MemberID == userID {
				f.memberships[w][i].IsActive = false
			}
		}
	}
	return nil
}
