package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// DraftLocks composes the existing write transaction and authorizer. Personal
// saves hold W(S), sorted P(S), then D(U), without entering an issue write gate.
type DraftLocks struct {
	base       Locks
	workspaces WorkspaceSharer
	projects   DraftProjects
	drafts     DraftReader
}

func NewDraftLocks(base Locks, w WorkspaceSharer, p DraftProjects, d DraftReader) DraftLocks {
	return DraftLocks{base: base, workspaces: w, projects: p, drafts: d}
}

type draftWrite struct {
	WorkspaceID uuid.UUID
	DraftID     *uuid.UUID
	SetProject  bool
	ProjectID   *uuid.UUID
}
type draftHeld struct {
	Draft     *domain.Draft
	ProjectID *uuid.UUID
}

func (l DraftLocks) seeWorkspace(ctx context.Context, actor shared.Actor, wid uuid.UUID) error {
	return l.base.seeWorkspace(ctx, actor, domain.ActionDraftWrite, wid)
}

// resource checks only existence. In particular the W-lock recheck must reveal
// a committed deletion before a demotion, without inspecting owner or chasing
// the current project's lock before workspace authorization.
func (l DraftLocks) resource(ctx context.Context, id uuid.UUID) (domain.Draft, error) {
	d, found, err := l.drafts.GetDraft(ctx, id)
	if err != nil {
		return domain.Draft{}, err
	}
	if !found {
		return domain.Draft{}, domain.ErrDraftNotFound
	}
	if d.ProjectID != nil {
		wid, found, err := l.base.projects.Find(ctx, *d.ProjectID)
		if err != nil {
			return domain.Draft{}, err
		}
		if !found || wid != d.WorkspaceID {
			return domain.Draft{}, domain.ErrDraftNotFound
		}
	}
	return d, nil
}
func (l DraftLocks) read(ctx context.Context, actor shared.Actor, id uuid.UUID) (domain.Draft, error) {
	d, err := l.resource(ctx, id)
	if err != nil {
		return domain.Draft{}, err
	}
	if err = l.seeWorkspace(ctx, actor, d.WorkspaceID); err != nil {
		if errors.Is(err, domain.ErrWorkspaceNotFound) {
			err = domain.ErrDraftNotFound
		}
		return domain.Draft{}, err
	}
	if d.CreatedByID != actor.UserID {
		return domain.Draft{}, domain.ErrDraftNotFound
	}
	return d, nil
}
func (l DraftLocks) allowProject(ctx context.Context, actor shared.Actor, wid, pid uuid.UUID) error {
	_, err := l.base.auth.Authorize(ctx, actor, shared.Action("project.read"), shared.Target{WorkspaceID: wid, ProjectID: pid})
	if errors.Is(err, shared.ErrNotVisible) {
		return draftProjectInvalid()
	}
	var e *shared.Error
	if errors.As(err, &e) && e.Kind == shared.KindForbidden {
		return draftProjectInvalid()
	}
	return err
}
func draftProjectInvalid() error {
	return shared.Invalid(shared.FieldError{Field: "project_id", Code: shared.FieldNotAllowed, Message: "must name an undeleted project you can see in this workspace"})
}
func sameDraftProject(a, b *uuid.UUID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func (l DraftLocks) run(ctx context.Context, actor shared.Actor, w draftWrite, fn func(context.Context, draftHeld) error) error {
	var before *domain.Draft
	if w.DraftID != nil {
		d, err := l.read(ctx, actor, *w.DraftID)
		if err != nil {
			return err
		}
		before = &d
		w.WorkspaceID = d.WorkspaceID
	} else if err := l.seeWorkspace(ctx, actor, w.WorkspaceID); err != nil {
		return err
	}
	if w.SetProject && w.ProjectID != nil {
		wid, found, err := l.base.projects.Find(ctx, *w.ProjectID)
		if err != nil {
			return err
		}
		if !found || wid != w.WorkspaceID {
			return draftProjectInvalid()
		}
		if err = l.allowProject(ctx, actor, w.WorkspaceID, *w.ProjectID); err != nil {
			return err
		}
	}
	ids := []uuid.UUID{}
	if before != nil && before.ProjectID != nil {
		ids = append(ids, *before.ProjectID)
	}
	if w.SetProject && w.ProjectID != nil {
		ids = append(ids, *w.ProjectID)
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return a.Compare(b) })
	ids = slices.Compact(ids)
	return l.base.tx.WithinTx(ctx, func(ctx context.Context) error {
		found, err := l.workspaces.ShareWorkspaceByID(ctx, w.WorkspaceID)
		if err != nil {
			return err
		}
		if !found {
			if before != nil {
				return domain.ErrDraftNotFound
			}
			return domain.ErrWorkspaceNotFound
		}
		if before != nil {
			if _, err := l.resource(ctx, *w.DraftID); err != nil {
				return err
			}
		}
		if err := l.seeWorkspace(ctx, actor, w.WorkspaceID); err != nil {
			if before != nil && errors.Is(err, domain.ErrWorkspaceNotFound) {
				return domain.ErrDraftNotFound
			}
			return err
		}
		locked := map[uuid.UUID]uuid.UUID{}
		if len(ids) > 0 {
			locked, err = l.projects.ShareMany(ctx, ids)
			if err != nil {
				return err
			}
		}
		h := draftHeld{ProjectID: w.ProjectID}
		if before != nil {
			if before.ProjectID != nil {
				if wid, ok := locked[*before.ProjectID]; !ok || wid != w.WorkspaceID {
					return domain.ErrDraftNotFound
				}
			}
			d, found, err := l.drafts.LockDraft(ctx, *w.DraftID)
			if err != nil {
				return err
			}
			if !found || d.WorkspaceID != w.WorkspaceID || d.CreatedByID != actor.UserID {
				return domain.ErrDraftNotFound
			}
			if !sameDraftProject(before.ProjectID, d.ProjectID) {
				return domain.ErrDraftChanged
			}
			h.Draft = &d
			if !w.SetProject {
				h.ProjectID = d.ProjectID
			}
		}
		if w.SetProject && w.ProjectID != nil {
			if wid, ok := locked[*w.ProjectID]; !ok || wid != w.WorkspaceID {
				return draftProjectInvalid()
			}
			if err := l.allowProject(ctx, actor, w.WorkspaceID, *w.ProjectID); err != nil {
				return err
			}
		}
		return fn(ctx, h)
	})
}
