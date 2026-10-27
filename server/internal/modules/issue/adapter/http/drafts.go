package httpadapter

import (
	"context"
	"encoding/json"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type ListDraftsUseCase interface {
	Execute(context.Context, string, *int, *string) (app.DraftPage, error)
}
type GetDraftUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.Draft, error)
}
type CreateDraftUseCase interface {
	Execute(context.Context, string, domain.NewDraft) (domain.Draft, error)
}
type UpdateDraftUseCase interface {
	Execute(context.Context, uuid.UUID, domain.DraftPatch) (domain.Draft, error)
}
type DeleteDraftUseCase interface {
	Execute(context.Context, uuid.UUID) error
}

func (h handler) ListDrafts(ctx context.Context, req gen.ListDraftsRequestObject) (gen.ListDraftsResponseObject, error) {
	if err := checkDraftQuery(rawListQuery(ctx)); err != nil {
		return nil, err
	}
	page, err := h.uc.ListDrafts.Execute(ctx, req.Slug, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	rows := make([]gen.Draft, 0, len(page.Data))
	for _, d := range page.Data {
		row, err := draftResponse(d)
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return gen.ListDrafts200JSONResponse{Data: rows, NextCursor: orNull(page.NextCursor), Total: page.Total}, nil
}
func (h handler) GetDraft(ctx context.Context, req gen.GetDraftRequestObject) (gen.GetDraftResponseObject, error) {
	d, err := h.uc.GetDraft.Execute(ctx, req.DraftID)
	if err != nil {
		return nil, err
	}
	row, err := draftResponse(d)
	return gen.GetDraft200JSONResponse(row), err
}
func (h handler) CreateDraft(ctx context.Context, req gen.CreateDraftRequestObject) (gen.CreateDraftResponseObject, error) {
	payload, err := json.Marshal(req.Body.Payload)
	if err != nil {
		return nil, err
	}
	_, project := given(req.Body.ProjectID)
	d, err := h.uc.CreateDraft.Execute(ctx, req.Slug, domain.NewDraft{ProjectID: project, Payload: payload})
	if err != nil {
		return nil, err
	}
	row, err := draftResponse(d)
	return gen.CreateDraft201JSONResponse(row), err
}
func (h handler) UpdateDraft(ctx context.Context, req gen.UpdateDraftRequestObject) (gen.UpdateDraftResponseObject, error) {
	in := domain.DraftPatch{SetPayload: req.Body.Payload != nil}
	in.SetProject, in.ProjectID = given(req.Body.ProjectID)
	if in.SetPayload {
		payload, err := json.Marshal(req.Body.Payload)
		if err != nil {
			return nil, err
		}
		in.Payload = payload
	}
	d, err := h.uc.UpdateDraft.Execute(ctx, req.DraftID, in)
	if err != nil {
		return nil, err
	}
	row, err := draftResponse(d)
	return gen.UpdateDraft200JSONResponse(row), err
}
func (h handler) DeleteDraft(ctx context.Context, req gen.DeleteDraftRequestObject) (gen.DeleteDraftResponseObject, error) {
	if err := h.uc.DeleteDraft.Execute(ctx, req.DraftID); err != nil {
		return nil, err
	}
	return gen.DeleteDraft204Response{}, nil
}
func draftResponse(d domain.Draft) (gen.Draft, error) {
	var payload gen.IssueDraftPayload
	if err := json.Unmarshal(d.Payload, &payload); err != nil {
		return gen.Draft{}, err
	}
	return gen.Draft{ID: d.ID, WorkspaceID: d.WorkspaceID, ProjectID: orNull(d.ProjectID), Payload: payload, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}, nil
}

// PublishDraftUseCase consumes only the stored draft.
type PublishDraftUseCase interface {
	Execute(context.Context, uuid.UUID) (domain.Issue, error)
}

func (h handler) PublishDraft(ctx context.Context, req gen.PublishDraftRequestObject) (gen.PublishDraftResponseObject, error) {
	i, err := h.uc.PublishDraft.Execute(ctx, req.DraftID)
	if err != nil {
		return nil, err
	}
	return gen.PublishDraft201JSONResponse(issueDetail(i)), nil
}
