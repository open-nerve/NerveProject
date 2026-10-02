package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// JoinProject makes the caller a member of a project he sees: POST
// /api/v0/projects/{project_id}/join (M3 design 3.5).
type JoinProject struct {
	locks    Locks
	projects MemberJoiner
	tx       shared.TxManager
	clock    Clock
}

// NewJoinProject returns the use case.
func NewJoinProject(locks Locks, projects MemberJoiner, tx shared.TxManager, clock Clock) *JoinProject {
	return &JoinProject{locks: locks, projects: projects, tx: tx, clock: clock}
}

// Execute, in one transaction, in the order of M3 design 3.6: the
// project's locks (Locks: its workspace FOR SHARE; the caller's membership
// of the workspace FOR SHARE, which comes before the project in the lock
// order, convention 3; the project FOR NO KEY UPDATE) and the decision on
// project.join, which asks only that he see it; his workspace role, admin
// or member by set, else forbidden, before his membership of the project
// is looked at, so that a workspace guest who is its member is refused as
// any guest (3.5); his membership of the project. An active member is
// left as he is. Otherwise the clock, then his ended membership restored
// with the lesser of its role and his workspace role (domain.JoinRole,
// convention 6), or a new one with his workspace role; his display
// settings at the default place unless he has them (3.18). The answer is
// the project as he sees it.
func (u *JoinProject) Execute(ctx context.Context, id uuid.UUID) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	var joined domain.Project
	err = u.tx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionJoin, targets: []uuid.UUID{actor.UserID}})
		if err != nil {
			return err
		}
		if !domain.CanJoin(h.grant.WorkspaceRole) {
			return shared.Forbidden()
		}
		memberships, err := u.projects.Memberships(ctx, id, []uuid.UUID{actor.UserID})
		if err != nil {
			return err
		}
		if m, ok := memberships[actor.UserID]; !ok || !m.Active {
			ended := endedOf(memberships, actor.UserID)
			var was *shared.Role
			if ended != nil {
				was = &ended.Role
			}
			if err := (growth{workspaceID: h.project.WorkspaceID, projectID: id, user: actor.UserID, ended: ended,
				role: domain.JoinRole(was, h.grant.WorkspaceRole), sortOrder: domain.DefaultSortOrder, by: actor.UserID,
				now: u.clock.Now()}).apply(ctx, u.projects); err != nil {
				return err
			}
		}
		joined, err = answer(ctx, u.projects, id, actor.UserID)
		return err
	})
	if err != nil {
		return domain.Project{}, err
	}
	return joined, nil
}
