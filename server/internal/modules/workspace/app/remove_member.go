package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RemoveWorkspaceMember removes a member from a workspace: DELETE
// /api/v0/workspace-members/{workspace_member_id}.
type RemoveWorkspaceMember struct {
	members MemberLocker
	end     membershipEnd
	auth    shared.Authorizer
	tx      shared.TxManager
	clock   Clock
}

// NewRemoveWorkspaceMember returns the use case.
func NewRemoveWorkspaceMember(members MemberRemover, profiles MemberProfiles, projects ProjectCascade, auth shared.Authorizer,
	tx shared.TxManager, clock Clock) *RemoveWorkspaceMember {
	return &RemoveWorkspaceMember{members: members, end: membershipEnd{members: members, profiles: profiles, projects: projects}, auth: auth,
		tx: tx, clock: clock}
}

// Execute removes the membership id, in one transaction (M3 design 3.6's
// lock table): the membership read for its workspace, the workspace row FOR
// NO KEY UPDATE, the membership read again under the lock, the decision on
// workspace_member.remove; then the checks on the target, which only a
// caller allowed to remove members gets to see, in updateWorkspaceMember's
// order: an ended membership is workspace.member_not_found, the caller's
// own workspace.own_membership (he leaves through leaveWorkspace); then the
// clock, read under the lock (3.3), and the ending: the workspace's pending
// invitation to his address, his membership, then his memberships of its
// projects (membershipEnd). Were he the only admin of a project with other
// members, project.sole_admin (3.7 rule 2), and the whole removal rolls
// back.
func (u *RemoveWorkspaceMember) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		m, err := lockedMember(ctx, u.members, id)
		if err != nil {
			return err
		}
		if _, err := decide(ctx, u.auth, actor, domain.ActionMemberRemove, m.WorkspaceID, domain.ErrMemberNotFound); err != nil {
			return err
		}
		switch {
		case !m.IsActive:
			return domain.ErrMemberNotFound
		case m.MemberID == actor.UserID:
			return domain.ErrOwnMembership
		}
		return u.end.run(ctx, m.WorkspaceID, m.MemberID, actor.UserID, u.clock.Now())
	})
}
