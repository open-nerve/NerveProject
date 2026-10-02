package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LeaveWorkspace ends the caller's own membership of a workspace: POST
// /api/v0/workspaces/{slug}/leave.
type LeaveWorkspace struct {
	workspaces WorkspaceLeaver
	end        membershipEnd
	auth       shared.Authorizer
	tx         shared.TxManager
	clock      Clock
}

// NewLeaveWorkspace returns the use case.
func NewLeaveWorkspace(workspaces WorkspaceLeaver, profiles MemberProfiles, projects ProjectCascade, auth shared.Authorizer, tx shared.TxManager,
	clock Clock) *LeaveWorkspace {
	return &LeaveWorkspace{workspaces: workspaces, end: membershipEnd{members: workspaces, profiles: profiles, projects: projects}, auth: auth,
		tx: tx, clock: clock}
}

// Execute ends the caller's membership of the workspace slug, in one
// transaction (M3 design 3.6's lock table): the workspace row FOR NO KEY
// UPDATE, then the decision on workspace.leave, which every active member
// may take; then, the caller being its admin, whether the workspace has
// another active admin: if not, workspace.sole_admin, also when he is its
// only member (3.7 rule 1); then the clock, read under the lock (3.3), and
// the ending, by himself: the workspace's pending invitation to his
// address, his membership, then his memberships of its projects
// (membershipEnd). Were he the only admin of a project with other members,
// project.sole_admin (3.7 rule 2), and the whole leaving rolls back.
func (u *LeaveWorkspace) Execute(ctx context.Context, slug string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		id, grant, err := lockAndDecide(ctx, u.workspaces.LockWorkspaceBySlug, u.auth, actor, slug, domain.ActionLeave)
		if err != nil {
			return err
		}
		if grant.WorkspaceRole == shared.RoleAdmin {
			other, err := u.workspaces.HasOtherAdmin(ctx, id, actor.UserID)
			switch {
			case err != nil:
				return err
			case !other:
				return domain.ErrSoleAdmin
			}
		}
		return u.end.run(ctx, id, actor.UserID, actor.UserID, u.clock.Now())
	})
}
