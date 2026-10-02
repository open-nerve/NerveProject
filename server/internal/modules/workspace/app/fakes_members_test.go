package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The members' repositories of fakeWorkspaces: MemberLister and
// MemberUpdater.

func (f *fakeWorkspaces) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	f.log.add(ctx, "ListMembers %s", workspaceID)
	if f.membersErr != nil {
		return nil, fmt.Errorf("list workspace members: %w", f.membersErr)
	}
	return f.memberships[workspaceID], nil
}

func (f *fakeWorkspaces) MemberByID(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	f.log.add(ctx, "MemberByID %s", id)
	if f.membersErr != nil {
		return domain.Membership{}, fmt.Errorf("read workspace member: %w", f.membersErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			return list[i], nil
		}
	}
	return domain.Membership{}, app.ErrNotFound
}

// LockWorkspace answers as the slug locks do, for the workspace with id.
func (f *fakeWorkspaces) LockWorkspace(ctx context.Context, id uuid.UUID) error {
	f.log.add(ctx, "LockWorkspace %s", id)
	return f.lockByID(id)
}

// lockByID answers as lock does, for the workspace with id; once locked, it
// runs onLock.
func (f *fakeWorkspaces) lockByID(id uuid.UUID) error {
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return app.ErrNotFound
	}
	if _, err := f.lock(f.workspaces[i].Slug); err != nil {
		return err
	}
	if f.onLock != nil {
		f.onLock()
	}
	return nil
}

func (f *fakeWorkspaces) UpdateMemberRole(ctx context.Context, id uuid.UUID, role shared.Role, by uuid.UUID, now time.Time) (domain.Membership, error) {
	f.log.add(ctx, "UpdateMemberRole %s to %d by %s at %s", id, role, by, now.Format(time.RFC3339Nano))
	if f.roleErr != nil {
		return domain.Membership{}, fmt.Errorf("update workspace member: %w", f.roleErr)
	}
	for _, list := range f.memberships {
		if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.ID == id }); i >= 0 {
			list[i].Role = role
			return list[i], nil
		}
	}
	return domain.Membership{}, fmt.Errorf("update workspace member %s: no such row", id)
}

// The ending's statements (app.MembershipEnder): each logs its call and
// fails with the error set for it; EndMember also ends the membership the
// fake holds, as the store ends the row.

func (f *fakeWorkspaces) DeletePendingInvitations(ctx context.Context, workspaceID uuid.UUID, email string, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeletePendingInvitations %s %s by %s at %s", workspaceID, email, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["DeletePendingInvitations"]; err != nil {
		return fmt.Errorf("delete the pending invitations: %w", err)
	}
	return nil
}

func (f *fakeWorkspaces) EndMember(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndMember %s %s by %s at %s", workspaceID, userID, by, now.Format(time.RFC3339Nano))
	if err := f.endErrs["EndMember"]; err != nil {
		return fmt.Errorf("end workspace member: %w", err)
	}
	list := f.memberships[workspaceID]
	if i := slices.IndexFunc(list, func(m domain.Membership) bool { return m.MemberID == userID }); i >= 0 {
		list[i].IsActive = false
		return nil
	}
	return fmt.Errorf("end workspace member %s of %s: no such row", userID, workspaceID)
}

// HasOtherAdmin answers whether the workspace's memberships it holds have
// an active admin other than userID.
func (f *fakeWorkspaces) HasOtherAdmin(ctx context.Context, workspaceID, userID uuid.UUID) (bool, error) {
	f.log.add(ctx, "HasOtherAdmin %s %s", workspaceID, userID)
	if err := f.endErrs["HasOtherAdmin"]; err != nil {
		return false, fmt.Errorf("look for another admin: %w", err)
	}
	return slices.ContainsFunc(f.memberships[workspaceID], func(m domain.Membership) bool {
		return m.MemberID != userID && m.Role == shared.RoleAdmin && m.IsActive
	}), nil
}
