package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProjectMember changes a project member's role: PATCH
// /api/v0/project-members/{project_member_id} (M3 design 3.5).
type UpdateProjectMember struct {
	locks   Locks
	members MemberRoleChanger
	tx      shared.TxManager
	clock   Clock
}

// NewUpdateProjectMember returns the use case.
func NewUpdateProjectMember(locks Locks, members MemberRoleChanger, tx shared.TxManager, clock Clock) *UpdateProjectMember {
	return &UpdateProjectMember{locks: locks, members: members, tx: tx, clock: clock}
}

// Execute checks role, then in one transaction, in the order of M3 design
// 3.6: the membership's locks (lockRowAndDecide: the membership read for
// its project and workspace, the workspace FOR SHARE, its member's
// membership of the workspace FOR SHARE, the project FOR NO KEY UPDATE, the
// membership read again) and the decision on project_member.update; then
// the checks that only a caller allowed to change roles gets to see: an
// ended membership is project.member_not_found; then 3.5's rules
// (domain.CheckRoleChange) on the caller's grant, whether the membership is
// his own, the member's role, his workspace role as locked, and the role
// given; then the change, at the time the clock gives under the locks. The
// answer is the membership as stored.
func (u *UpdateProjectMember) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Member{}, err
	}
	if err := domain.CheckMemberRole(role); err != nil {
		return domain.Member{}, err
	}
	var updated domain.Member
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, m, err := lockRowAndDecide(ctx, u.locks, actor, rowWrite[ProjectMembership]{id: id, action: domain.ActionMemberUpdate,
			find: u.members.MemberByID, targets: memberOf, notFound: domain.ErrMemberNotFound})
		if err != nil {
			return err
		}
		if !m.Active {
			return domain.ErrMemberNotFound
		}
		workspaceRole, ok := h.roles[m.MemberID]
		if !ok {
			// Every shrinking of a workspace membership ends its project
			// memberships (3.6 convention 6): its absence is a bug.
			return fmt.Errorf("project membership %s: %s is no active member of the workspace", m.ID, m.MemberID)
		}
		if err := domain.CheckRoleChange(domain.RoleChange{Caller: h.grant, Own: m.MemberID == actor.UserID, From: m.Role,
			WorkspaceRole: workspaceRole, To: role}); err != nil {
			return err
		}
		if updated, err = u.members.UpdateMemberRole(ctx, m.ID, role, actor.UserID, u.clock.Now()); err != nil {
			return err
		}
		if updated.ID != m.ID {
			return fmt.Errorf("project membership %s changed as %s", m.ID, updated.ID)
		}
		return nil
	})
	if err != nil {
		return domain.Member{}, err
	}
	return updated, nil
}

// memberOf is the account of m: the target of a change of his role, whose
// membership of the workspace the change locks (convention 3).
func memberOf(m ProjectMembership) []uuid.UUID {
	return []uuid.UUID{m.MemberID}
}
