package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type publishHeld struct {
	Draft domain.Draft
	Issue held
}

// publish enters the project gate without a connection, then holds W(S),
// the initially observed P(N), and D(U) in one transaction. A changed project
// is a conflict, never a reason to chase a second project under D(U).
func (l DraftLocks) publish(ctx context.Context, actor shared.Actor, id uuid.UUID, fn func(context.Context, publishHeld) error) error {
	before, err := l.read(ctx, actor, id)
	if err != nil {
		return err
	}
	if before.ProjectID == nil {
		return shared.Invalid(shared.FieldError{Field: "project_id", Code: shared.FieldRequired, Message: "is required"})
	}
	projectID := *before.ProjectID
	workspaceID, found, err := l.base.projects.Find(ctx, projectID)
	if err != nil {
		return err
	}
	// Resource disappearance conceals the draft; authorization invisibility
	// retains create's project404. A shared notFound cannot represent both.
	if !found || workspaceID != before.WorkspaceID {
		return domain.ErrDraftNotFound
	}
	if _, err = l.base.decide(ctx, actor, domain.ActionCreate, workspaceID, projectID, nil, domain.ErrProjectNotFound); err != nil {
		return err
	}
	leave, err := l.base.gate.Enter(ctx, projectID)
	if err != nil {
		return err
	}
	defer leave()
	return l.base.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := l.workspaces.ShareWorkspaceByID(ctx, before.WorkspaceID)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrDraftNotFound
		}
		if _, err = l.resource(ctx, id); err != nil {
			return err
		}
		if err = l.seeWorkspace(ctx, actor, before.WorkspaceID); err != nil {
			if errors.Is(err, domain.ErrWorkspaceNotFound) {
				return domain.ErrDraftNotFound
			}
			return err
		}
		p, found, err := l.base.projects.Lock(ctx, projectID)
		if err != nil {
			return err
		}
		if !found || p.WorkspaceID != before.WorkspaceID {
			return domain.ErrDraftNotFound
		}
		h := held{project: p}
		h.grant, err = l.base.decide(ctx, actor, domain.ActionCreate, p.WorkspaceID, projectID, nil, domain.ErrProjectNotFound)
		if err != nil {
			return err
		}
		d, found, err := l.drafts.LockDraft(ctx, id)
		if err != nil {
			return err
		}
		if !found || d.WorkspaceID != before.WorkspaceID || d.CreatedByID != actor.UserID {
			return domain.ErrDraftNotFound
		}
		if !sameDraftProject(before.ProjectID, d.ProjectID) {
			return domain.ErrDraftChanged
		}
		return fn(ctx, publishHeld{Draft: d, Issue: h})
	})
}
