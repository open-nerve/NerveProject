package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type PreparationHostReader interface {
	Inspect(context.Context, shared.Actor, HostRequest) (Host, error)
}
type DraftPreparationQuery struct {
	ActorID uuid.UUID
	Host    Host
	After   uuid.UUID
	Limit   int
}
type DraftPreparationReader interface {
	ListDraftPreparations(context.Context, DraftPreparationQuery) ([]domain.Asset, error)
}
type DraftPreparationPage struct {
	Data       []domain.Asset
	NextCursor *string
}
type ListDraftPreparationsDeps struct {
	Rows   DraftPreparationReader
	Hosts  PreparationHostReader
	ReadTx shared.TxManager
}
type ListDraftPreparations struct{ d ListDraftPreparationsDeps }

func NewListDraftPreparations(d ListDraftPreparationsDeps) *ListDraftPreparations {
	return &ListDraftPreparations{d: d}
}
func (u *ListDraftPreparations) Execute(ctx context.Context, draftID uuid.UUID, limit *int, cursor *string) (DraftPreparationPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return DraftPreparationPage{}, err
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return DraftPreparationPage{}, err
	}
	after, err := domain.DecodeDraftPreparationCursor(draftID, cursor)
	if err != nil {
		return DraftPreparationPage{}, err
	}
	if draftID == uuid.Nil() {
		return DraftPreparationPage{}, domain.ErrAssetNotFound
	}
	page := DraftPreparationPage{Data: []domain.Asset{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		request := HostRequest{Scope: domain.Scope{Kind: domain.ScopeDraft, ID: draftID}, Intent: ReadAsset}
		host, err := u.d.Hosts.Inspect(ctx, actor, request)
		if err != nil {
			// This private list has no public portion. Keep the owner's error
			// policy on all existing asset/draft operations unchanged.
			var problem *shared.Error
			if errors.As(err, &problem) && problem.Kind == shared.KindForbidden {
				return domain.ErrAssetNotFound
			}
			return err
		}
		if host.Scope != request.Scope || host.DraftID != draftID {
			return domain.ErrAssetScopeChanged
		}
		rows, err := u.d.Rows.ListDraftPreparations(ctx, DraftPreparationQuery{ActorID: actor.UserID, Host: host, After: after, Limit: size + 1})
		if err != nil {
			return err
		}
		if len(rows) > size {
			value, err := domain.EncodeDraftPreparationCursor(draftID, rows[size-1].ID)
			if err != nil {
				return err
			}
			page.NextCursor = &value
			rows = rows[:size]
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return DraftPreparationPage{}, err
	}
	return page, nil
}
