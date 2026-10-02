package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AddMembersDeps are addProjectMembers' ports.
type AddMembersDeps struct {
	Locks    Locks
	Projects MemberAdder
	Tx       shared.TxManager
	Clock    Clock
}

// AddProjectMembers adds workspace members to a project: POST
// /api/v0/projects/{project_id}/members.
type AddProjectMembers struct {
	d AddMembersDeps
}

// NewAddProjectMembers returns the use case.
func NewAddProjectMembers(d AddMembersDeps) *AddProjectMembers {
	return &AddProjectMembers{d: d}
}

// Execute checks the request, then in one transaction, in the order of M3
// design 3.6: the project's locks (Locks: its workspace FOR SHARE; the
// targets' memberships of the workspace FOR SHARE, which come before the
// project in the lock order, convention 3; the project FOR NO KEY UPDATE)
// and the decision on project_member.add; the targets' memberships of the
// project; the targets' check (domain.CheckTargets), after the decision, so
// that who may not add learns nothing of them; the clock; then each target,
// in the request's order: an ended membership restored with the role asked
// for, as the admin decides it now, or a new one; his display settings
// unless he has them, before his other projects in his sidebar (3.18). The
// answer is the targets' memberships as stored, in the request's order.
func (u *AddProjectMembers) Execute(ctx context.Context, projectID uuid.UUID, in []domain.NewMember) ([]domain.Member, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	if err := domain.CheckNewMembers(in); err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(in))
	for i, m := range in {
		ids[i] = m.MemberID
	}
	var added []domain.Member
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.d.Locks.lockAndDecide(ctx, actor, write{project: projectID, action: domain.ActionMemberAdd, targets: ids})
		if err != nil {
			return err
		}
		memberships, err := u.d.Projects.Memberships(ctx, projectID, ids)
		if err != nil {
			return err
		}
		targets := make([]domain.Target, len(in))
		for i, m := range in {
			targets[i] = domain.Target{NewMember: m, Member: memberships[m.MemberID].Active}
			if role, ok := h.roles[m.MemberID]; ok {
				targets[i].WorkspaceRole = &role
			}
		}
		if err := domain.CheckTargets(targets); err != nil {
			return err
		}
		now := u.d.Clock.Now()
		for _, m := range in {
			lowest, err := u.d.Projects.LowestSortOrder(ctx, h.project.WorkspaceID, m.MemberID)
			if err != nil {
				return err
			}
			if err := (growth{workspaceID: h.project.WorkspaceID, projectID: projectID, user: m.MemberID, ended: endedOf(memberships, m.MemberID),
				role: m.Role, sortOrder: domain.SortOrderFirst(lowest), by: actor.UserID, now: now}).apply(ctx, u.d.Projects); err != nil {
				return err
			}
		}
		added, err = storedMembers(ctx, u.d.Projects, projectID, ids)
		return err
	})
	if err != nil {
		return nil, err
	}
	return added, nil
}

// storedMembers are ids' active memberships of the project as stored, in
// ids' order; one missing is an internal error.
func storedMembers(ctx context.Context, members MemberLister, projectID uuid.UUID, ids []uuid.UUID) ([]domain.Member, error) {
	list, err := members.ListMembers(ctx, projectID)
	if err != nil {
		return nil, err
	}
	byAccount := make(map[uuid.UUID]domain.Member, len(list))
	for _, m := range list {
		byAccount[m.MemberID] = m
	}
	out := make([]domain.Member, len(ids))
	for i, id := range ids {
		m, ok := byAccount[id]
		if !ok {
			return nil, fmt.Errorf("the membership of %s in project %s is not there after its write", id, projectID)
		}
		out[i] = m
	}
	return out, nil
}
