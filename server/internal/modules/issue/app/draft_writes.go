package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type CreateDraftDeps struct {
	Locks      DraftLocks
	Workspaces WorkspaceDirectory
	Drafts     DraftWriter
	Clock      Clock
}
type CreateDraft struct{ d CreateDraftDeps }

func NewCreateDraft(d CreateDraftDeps) *CreateDraft { return &CreateDraft{d: d} }
func (u *CreateDraft) Execute(ctx context.Context, slug string, in domain.NewDraft) (domain.Draft, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Draft{}, err
	}
	if err := domain.CheckDraftPayload(in.Payload); err != nil {
		return domain.Draft{}, err
	}
	wid, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
	if err != nil {
		return domain.Draft{}, err
	}
	if !found {
		return domain.Draft{}, domain.ErrWorkspaceNotFound
	}
	var out domain.Draft
	err = u.d.Locks.run(ctx, actor, draftWrite{WorkspaceID: wid, SetProject: true, ProjectID: in.ProjectID}, func(ctx context.Context, h draftHeld) error {
		var err error
		out, err = u.d.Drafts.InsertDraft(ctx, DraftWrite{ID: uuid.NewV7(), WorkspaceID: wid, By: actor.UserID, ProjectID: h.ProjectID, Payload: in.Payload, At: u.d.Clock.Now()})
		return err
	})
	if err != nil {
		return domain.Draft{}, err
	}
	return out, nil
}

type UpdateDraftDeps struct {
	Locks  DraftLocks
	Drafts DraftWriter
	Clock  Clock
}
type UpdateDraft struct{ d UpdateDraftDeps }

func NewUpdateDraft(d UpdateDraftDeps) *UpdateDraft { return &UpdateDraft{d: d} }
func (u *UpdateDraft) Execute(ctx context.Context, id uuid.UUID, in domain.DraftPatch) (domain.Draft, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Draft{}, err
	}
	if in.SetPayload {
		if err := domain.CheckDraftPayload(in.Payload); err != nil {
			return domain.Draft{}, err
		}
	}
	var out domain.Draft
	err = u.d.Locks.run(ctx, actor, draftWrite{DraftID: &id, SetProject: in.SetProject, ProjectID: in.ProjectID}, func(ctx context.Context, h draftHeld) error {
		payload := h.Draft.Payload
		if in.SetPayload {
			payload = in.Payload
		}
		var err error
		out, err = u.d.Drafts.UpdateDraft(ctx, DraftWrite{ID: id, WorkspaceID: h.Draft.WorkspaceID, By: actor.UserID, ProjectID: h.ProjectID, Payload: payload, At: u.d.Clock.Now()})
		return err
	})
	if err != nil {
		return domain.Draft{}, err
	}
	return out, nil
}

type DeleteDraftDeps struct {
	Locks  DraftLocks
	Drafts DraftWriter
	Clock  Clock
}
type DeleteDraft struct{ d DeleteDraftDeps }

func NewDeleteDraft(d DeleteDraftDeps) *DeleteDraft { return &DeleteDraft{d: d} }
func (u *DeleteDraft) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, draftWrite{DraftID: &id}, func(ctx context.Context, _ draftHeld) error {
		return u.d.Drafts.DeleteDraft(ctx, DraftDeletion{ID: id, By: actor.UserID, At: u.d.Clock.Now()})
	})
}
