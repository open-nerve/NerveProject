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
