package app

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspaceMember changes a member's role: PATCH
// /api/v0/workspace-members/{workspace_member_id}.
type UpdateWorkspaceMember struct {
	members  MemberUpdater
	projects ProjectCascade
	profiles MemberProfiles
	auth     shared.Authorizer
	tx       shared.TxManager
	clock    Clock
}

// NewUpdateWorkspaceMember returns the use case.
func NewUpdateWorkspaceMember(members MemberUpdater, projects ProjectCascade, profiles MemberProfiles, auth shared.Authorizer,
	tx shared.TxManager, clock Clock) *UpdateWorkspaceMember {
	return &UpdateWorkspaceMember{members: members, projects: projects, profiles: profiles, auth: auth, tx: tx, clock: clock}
}

// Execute checks role, then in one transaction (M3 design 3.6): the
// membership read for its workspace, the workspace row FOR NO KEY UPDATE,
// the membership read again under the lock, the decision on
// workspace_member.update, then the checks on the target, which only a
// caller allowed to change roles gets to see: an ended membership is
// workspace.member_not_found, the caller's own workspace.own_membership;
// then the change, at the time the clock gives under the lock. A change to
// guest then makes him a guest in each of the workspace's projects he has a
// membership of, ended ones too (ProjectCascade.DemoteToGuest, M3 design
// 3.3), at the same time; a failure there rolls the change back.
// The answer carries the member's profile, read without a lock (M3 design
// 3.6 convention 1) before the commit, so a failed read changes nothing.
func (u *UpdateWorkspaceMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Member{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.Member{}, err
	}
	var updated domain.Member
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		m, err := u.lockedMember(ctx, id)
		if err != nil {
			return err
		}
		grant, err := decide(ctx, u.auth, actor, domain.ActionMemberUpdate, m.WorkspaceID, domain.ErrMemberNotFound)
		if err != nil {
			return err
		}
		switch {
		case !m.IsActive:
			return domain.ErrMemberNotFound
		case m.MemberID == actor.UserID:
			return domain.ErrOwnMembership
		}
		now := u.clock.Now()
		if m, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, now); err != nil {
			return err
		}
		if role == shared.RoleGuest {
			if err := u.projects.DemoteToGuest(ctx, m.WorkspaceID, m.MemberID, actor.UserID, now); err != nil {
				return err
			}
		}
		updated, err = u.withProfile(ctx, m, grant.WorkspaceRole)
		return err
	})
	if err != nil {
		return domain.Member{}, err
	}
	return updated, nil
}

// lockedMember reads the membership id, locks its workspace, and reads it
// again under the lock, which must still be of the workspace locked (M3
// design 3.6 convention 2): a membership or workspace deleted meanwhile,
// or a membership no longer of that workspace, is domain.ErrMemberNotFound.
func (u *UpdateWorkspaceMember) lockedMember(ctx context.Context, id uuid.UUID) (domain.Membership, error) {
	m, err := u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if err := u.members.LockWorkspace(ctx, m.WorkspaceID); err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	locked, err := u.members.MemberByID(ctx, id)
	if err != nil {
		return domain.Membership{}, memberNotFound(err)
	}
	if locked.WorkspaceID != m.WorkspaceID {
		return domain.Membership{}, domain.ErrMemberNotFound
	}
	return locked, nil
}

// memberNotFound turns ErrNotFound into domain.ErrMemberNotFound.
func memberNotFound(err error) error {
	if errors.Is(err, ErrNotFound) {
		return domain.ErrMemberNotFound
	}
	return err
}

// withProfile is m with its member's public profile, the address as a
// caller of role sees it.
func (u *UpdateWorkspaceMember) withProfile(ctx context.Context, m domain.Membership, role shared.Role) (domain.Member, error) {
	profiles, err := u.profiles.PublicProfiles(ctx, []uuid.UUID{m.MemberID})
	if err != nil {
		return domain.Member{}, err
	}
	if len(profiles) != 1 || profiles[0].ID != m.MemberID {
		// The foreign key keeps every member's account: its absence is a bug.
		return domain.Member{}, fmt.Errorf("workspace member %s: no account %s", m.ID, m.MemberID)
	}
	return domain.Member{Membership: m, User: memberUser(profiles[0], domain.SeesEmails(role))}, nil
}
