package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LeaveProject ends the caller's own project membership: POST
// /api/v0/projects/{project_id}/leave (M3 design 3.7).
type LeaveProject struct {
	locks   Locks
	members MemberLeaver
	tx      shared.TxManager
	clock   Clock
}

// NewLeaveProject returns the use case.
func NewLeaveProject(locks Locks, members MemberLeaver, tx shared.TxManager, clock Clock) *LeaveProject {
	return &LeaveProject{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the project's
// locks (Locks.lockAndDecide: the workspace FOR SHARE, the project FOR NO
// KEY UPDATE) and the decision on project.leave, which only an active
// member of the project passes; then 3.7's rule 1: a project admin leaves
// only while the project has another active admin, also when he is its
// only member; then his membership ended, its row and its role kept, by
// himself at the time the clock gives under the locks. Every write of the
// project's memberships holds the project's row, so no other admin leaves
// or is removed between the read of the others and the ending.
func (u *LeaveProject) Execute(ctx context.Context, projectID uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionLeave})
		if err != nil {
			return err
		}
		if h.grant.ProjectRole == shared.RoleAdmin {
			other, err := u.members.HasOtherAdmin(ctx, projectID, actor.UserID)
			switch {
			case err != nil:
				return err
			case !other:
				return domain.ErrSoleAdmin
			}
		}
		return u.members.EndMember(ctx, projectID, actor.UserID, actor.UserID, u.clock.Now())
	})
}
