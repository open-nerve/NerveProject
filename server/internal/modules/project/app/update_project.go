package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateProject changes a project: PATCH /api/v0/projects/{project_id} (M3
// design 3.19).
type UpdateProjectDeps struct {
	Projects ProjectUpdater
	Locks    Locks
	Gate     ImageGate
	Caller   CallerLock
	Bindings ImageBindings
	Images   ImageReader
	Tx       shared.TxManager
	Clock    Clock
}
type UpdateProject struct{ d UpdateProjectDeps }

func NewUpdateProject(d UpdateProjectDeps) *UpdateProject { return &UpdateProject{d: d} }

// Execute checks p (domain.CheckProjectPatch), then, in one transaction, in
// the order of M3 design 3.6: the project's locks (Locks: its workspace FOR
// SHARE, then the project FOR NO KEY UPDATE) and the decision on
// project.update; an archived project refused (409 project.archived,
// 3.19); the lead and the default assignee p names checked, after the
// decision, so that a caller who may not change the project learns nothing
// of them (422 not_allowed unless an active member of the project who is
// not its guest, 3.19); the change, at the clock read under the locks, so
// a change that waited for another is not stamped earlier than it. The
// answer is the project as stored, as the caller sees it.
func (u *UpdateProject) Execute(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	if p, err = domain.CheckProjectPatch(p); err != nil {
		return domain.Project{}, err
	}
	if err := findAndDecide(ctx, u.d.Locks.projects, u.d.Locks.auth, actor, id, domain.ActionUpdate); err != nil {
		return domain.Project{}, err
	}
	leave, err := u.d.Gate.Enter(ctx, id)
	if err != nil {
		return domain.Project{}, err
	}
	defer leave()
	var updated domain.Project
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := u.d.Caller.LockCaller(ctx, actor, u.d.Clock.Now()); err != nil {
			return err
		}
		h, err := u.d.Locks.lockAndDecide(ctx, actor, write{project: id, action: domain.ActionUpdate})
		switch {
		case err != nil:
			return err
		case h.project.Archived:
			return domain.ErrArchived
		}
		if err := u.checkAssignees(ctx, id, p); err != nil {
			return err
		}
		current, err := answer(ctx, u.d.Projects, id, actor.UserID)
		if err != nil {
			return err
		}
		change := CoverImageChange{ActorID: actor.UserID, WorkspaceID: h.project.WorkspaceID, ProjectID: id, Current: current.CoverImageAssetID, Next: current.CoverImageAssetID}
		if p.SetCoverImage {
			change.Next = p.CoverImageAssetID
		}
		at, err := u.d.Bindings.LockImageBindings(ctx, change)
		if err != nil {
			return err
		}
		if err := u.d.Projects.UpdateProject(ctx, id, p, actor.UserID, at); err != nil {
			return err
		}
		if err := u.d.Bindings.ApplyImageBindings(ctx, change, at); err != nil {
			return err
		}
		updated, err = answer(ctx, u.d.Projects, id, actor.UserID)
		if err != nil {
			return err
		}
		images, err := withProjectImages(ctx, u.d.Images, []domain.Project{updated})
		if err != nil {
			return err
		}
		updated = images[0]
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return updated, nil
}

// checkAssignees refuses a lead or a default assignee that p sets to an
// account who is not an active member of the project, or is its guest:
// one 422 naming each such field.
func (u *UpdateProject) checkAssignees(ctx context.Context, id uuid.UUID, p domain.ProjectPatch) error {
	fields := []struct {
		name    string
		set     bool
		account *uuid.UUID
	}{{"project_lead_id", p.SetLead, p.LeadID}, {"default_assignee_id", p.SetDefaultAssignee, p.DefaultAssigneeID}}
	var accounts []uuid.UUID
	for _, f := range fields {
		if f.set && f.account != nil {
			accounts = append(accounts, *f.account)
		}
	}
	if len(accounts) == 0 {
		return nil
	}
	members, err := u.d.Projects.Memberships(ctx, id, accounts)
	if err != nil {
		return err
	}
	var problems []shared.FieldError
	for _, f := range fields {
		if !f.set || f.account == nil {
			continue
		}
		if m, ok := members[*f.account]; !ok || !m.Active || !domain.CanAssign(m.Role) {
			problems = append(problems, domain.Unassignable(f.name))
		}
	}
	if len(problems) > 0 {
		return shared.Invalid(problems...)
	}
	return nil
}
