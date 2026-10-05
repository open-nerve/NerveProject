package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RemoveProjectMember ends another member's project membership: DELETE
// /api/v0/project-members/{project_member_id} (M3 design 3.5).
type RemoveProjectMember struct {
	locks   Locks
	members MemberRemover
	tx      shared.TxManager
	clock   Clock
}

// NewRemoveProjectMember returns the use case.
func NewRemoveProjectMember(locks Locks, members MemberRemover, tx shared.TxManager, clock Clock) *RemoveProjectMember {
	return &RemoveProjectMember{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the
// membership's locks (lockRowAndDecide: the membership read for its
// project and workspace, the workspace FOR SHARE, the project FOR NO KEY
// UPDATE, the membership read again) and the decision on
// project_member.remove; then the checks that only a caller allowed to
// remove members gets to see: an ended membership is
// project.member_not_found; then 3.5's rule (domain.CheckRemoval): his own
// membership, and one whose role is above his project role, are refused;
// then the membership ended, its row and its role kept, by the caller at
// the time the clock gives under the locks. A caller who may remove members
// is a project admin, or a project member who is the workspace's admin and
// removes no one above his own role: the project keeps an admin.
func (u *RemoveProjectMember) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, m, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[ProjectMembership]{id: id, action: domain.ActionMemberRemove,
			find: u.members.MemberByID, notFound: domain.ErrMemberNotFound})
		if err != nil {
			return err
		}
		if !m.Active {
			return domain.ErrMemberNotFound
		}
		if err := domain.CheckRemoval(h.grant.ProjectRole, m.MemberID == actor.UserID, m.Role); err != nil {
			return err
		}
		return u.members.EndMember(ctx, m.ProjectID, m.MemberID, actor.UserID, u.clock.Now())
	})
}
