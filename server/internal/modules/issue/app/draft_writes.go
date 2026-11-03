package app

import (
	"context"
	"encoding/json"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type CreateDraftDeps struct {
	Locks      DraftLocks
	Workspaces WorkspaceDirectory
	Drafts     DraftWriter
	Clock      Clock
	Sanitizer  HTMLSanitizer
	Transfer   DraftAssetTransfer
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
		images, err := draftImageIDs(in.Payload, u.d.Sanitizer)
		if err != nil {
			return err
		}
		id := uuid.NewV7()
		q := DraftAssetRequest{ActorID: actor.UserID, WorkspaceID: wid, DraftID: id, OldProjectID: h.ProjectID, NewProjectID: h.ProjectID, ImageIDs: images}
		var at time.Time
		if len(images) > 0 {
			at, err = u.d.Transfer.LockDraftAssets(ctx, q)
			if err != nil {
				return err
			}
		} else {
			at = u.d.Clock.Now()
		}
		out, err = u.d.Drafts.InsertDraft(ctx, DraftWrite{ID: id, WorkspaceID: wid, By: actor.UserID, ProjectID: h.ProjectID, Payload: in.Payload, At: at})
		if err != nil || len(images) == 0 {
			return err
		}
		return u.d.Transfer.MoveDraftAssets(ctx, q, uuid.Nil(), at)
	})
	if err != nil {
		return domain.Draft{}, err
	}
	return out, nil
}

type UpdateDraftDeps struct {
	Locks     DraftLocks
	Drafts    DraftWriter
	Sanitizer HTMLSanitizer
	Transfer  DraftAssetTransfer
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
		images, err := draftImageIDs(payload, u.d.Sanitizer)
		if err != nil {
			return err
		}
		q := DraftAssetRequest{ActorID: actor.UserID, WorkspaceID: h.Draft.WorkspaceID, DraftID: id, OldProjectID: h.Draft.ProjectID, NewProjectID: h.ProjectID, ImageIDs: images}
		at, err := u.d.Transfer.LockDraftAssets(ctx, q)
		if err != nil {
			return err
		}
		out, err = u.d.Drafts.UpdateDraft(ctx, DraftWrite{ID: id, WorkspaceID: h.Draft.WorkspaceID, By: actor.UserID, ProjectID: h.ProjectID, Payload: payload, At: at})
		if err != nil {
			return err
		}
		return u.d.Transfer.MoveDraftAssets(ctx, q, uuid.Nil(), at)
	})
	if err != nil {
		return domain.Draft{}, err
	}
	return out, nil
}

// Use the existing sanitizer only as a reference projection. A draft preserves
// its unfinished JSON, including business values rejected only on publication.
func draftImageIDs(raw json.RawMessage, sanitizer HTMLSanitizer) ([]uuid.UUID, error) {
	var payload struct {
		HTML string `json:"description_html"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return sanitizer.Clean(payload.HTML).Identity.AssetImageIDs(), nil
}

type DeleteDraftDeps struct {
	Locks  DraftLocks
	Drafts DraftWriter
	Assets DraftAssets
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
		deletion := DraftDeletion{ID: id, By: actor.UserID, At: u.d.Clock.Now()}
		if err := u.d.Drafts.DeleteDraft(ctx, deletion); err != nil {
			return err
		}
		return u.d.Assets.DeleteDraftAssets(ctx, deletion)
	})
}
