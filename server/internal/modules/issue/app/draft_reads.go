package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type DraftPage struct {
	Data       []domain.Draft
	NextCursor *string
	Total      int64
}
type ListDraftsDeps struct {
	ReadTx     shared.TxManager
	Locks      DraftLocks
	Workspaces WorkspaceDirectory
	Projects   DraftProjects
	Drafts     DraftLister
}
type ListDrafts struct{ d ListDraftsDeps }

func NewListDrafts(d ListDraftsDeps) *ListDrafts { return &ListDrafts{d: d} }
func (u *ListDrafts) Execute(ctx context.Context, slug string, limit *int, cursor *string) (DraftPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return DraftPage{}, err
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return DraftPage{}, err
	}
	var after *domain.DraftCursor
	if cursor != nil {
		c, err := domain.DecodeDraftCursor(*cursor)
		if err != nil {
			return DraftPage{}, err
		}
		after = &c
	}
	page := DraftPage{Data: []domain.Draft{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		wid, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrWorkspaceNotFound
		}
		if err = u.d.Locks.seeWorkspace(ctx, actor, wid); err != nil {
			return err
		}
		ids, err := u.d.Projects.UndeletedIDs(ctx, wid)
		if err != nil {
			return err
		}
		rows, total, err := u.d.Drafts.ListDrafts(ctx, DraftListRead{WorkspaceID: wid, ActorID: actor.UserID, ProjectIDs: ids, After: after, Limit: size + 1})
		if err != nil {
			return err
		}
		page.Total = total
		if len(rows) > size {
			rows = rows[:size]
			last := rows[len(rows)-1]
			c, err := domain.EncodeDraftCursor(domain.DraftCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			if err != nil {
				return err
			}
			page.NextCursor = &c
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return DraftPage{}, err
	}
	return page, nil
}

type GetDraftDeps struct {
	ReadTx shared.TxManager
	Locks  DraftLocks
}
type GetDraft struct{ d GetDraftDeps }

func NewGetDraft(d GetDraftDeps) *GetDraft { return &GetDraft{d: d} }
func (u *GetDraft) Execute(ctx context.Context, id uuid.UUID) (domain.Draft, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Draft{}, err
	}
	var out domain.Draft
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error { var err error; out, err = u.d.Locks.read(ctx, actor, id); return err })
	if err != nil {
		return domain.Draft{}, err
	}
	return out, nil
}
