package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateWorkspaceMember changes a member's role: PATCH
// /api/v0/workspace-members/{workspace_member_id}.
type UpdateWorkspaceMember struct {
	caller   CallerLock
	images   MemberAvatarReader
	members  MemberUpdater
	projects ProjectCascade
	profiles MemberProfiles
	auth     shared.Authorizer
	tx       shared.TxManager
	clock    Clock
}

type UpdateMemberDeps struct {
	Members  MemberUpdater
	Projects ProjectCascade
	Profiles MemberProfiles
	Auth     shared.Authorizer
	Caller   CallerLock
	Images   MemberAvatarReader
	Tx       shared.TxManager
	Clock    Clock
}

func NewUpdateWorkspaceMember(d UpdateMemberDeps) *UpdateWorkspaceMember {
	return &UpdateWorkspaceMember{members: d.Members, projects: d.Projects, profiles: d.Profiles, auth: d.Auth, caller: d.Caller, images: d.Images, tx: d.Tx, clock: d.Clock}
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
		if err := u.caller.LockCaller(ctx, actor, u.clock.Now()); err != nil {
			return err
		}
		m, err := lockedMember(ctx, u.members, id)
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
		if err != nil {
			return err
		}
		images, err := withMemberAvatars(ctx, u.images, []domain.Member{updated})
		if err != nil {
			return err
		}
		updated = images[0]
		return nil
	})
	if err != nil {
		return domain.Member{}, err
	}
	return updated, nil
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
