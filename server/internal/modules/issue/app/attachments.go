package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AttachmentAccess interface {
	Inspect(context.Context, shared.Actor, AssetRequest) (AssetHost, error)
}
type GetAttachmentsDeps struct {
	Access AttachmentAccess
	Rows   AttachmentReader
	ReadTx shared.TxManager
}
type GetAttachments struct {
	kind string
	d    GetAttachmentsDeps
}
type AttachmentPage struct {
	Data       []Attachment
	NextCursor string
}

func NewGetAttachments(kind string, d GetAttachmentsDeps) *GetAttachments {
	return &GetAttachments{kind: kind, d: d}
}

func (u *GetAttachments) Execute(ctx context.Context, id uuid.UUID, limit *int, cursor *string) (AttachmentPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return AttachmentPage{}, err
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return AttachmentPage{}, err
	}
	after := uuid.Nil()
	if cursor != nil {
		after, err = domain.DecodeAttachmentCursor(*cursor, u.kind, id)
		if err != nil {
			return AttachmentPage{}, err
		}
	}
	var kind AssetHostKind
	var missing error
	switch u.kind {
	case "issue":
		kind, missing = AssetIssue, domain.ErrIssueNotFound
	case "comment":
		kind, missing = AssetComment, domain.ErrCommentNotFound
	case "draft":
		kind, missing = AssetDraft, domain.ErrDraftNotFound
	default:
		return AttachmentPage{}, errors.New("invalid attachment host kind")
	}
	if id == uuid.Nil() {
		return AttachmentPage{}, missing
	}
	page := AttachmentPage{Data: []Attachment{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		h, err := u.d.Access.Inspect(ctx, actor, AssetRequest{ID: id, Kind: kind, Intent: AssetRead})
		if err != nil {
			return err
		}
		rows, err := u.d.Rows.ListAttachments(ctx, AttachmentRead{Host: ContentHost(h), After: after, Limit: size + 1})
		if err != nil {
			return err
		}
		if len(rows) > size+1 {
			return errors.New("attachment reader exceeded page limit")
		}
		previous := after
		for _, row := range rows {
			if row.ID.Compare(previous) <= 0 {
				return errors.New("attachment reader returned an invalid order")
			}
			previous = row.ID
		}
		if len(rows) > size {
			rows = rows[:size]
			page.NextCursor, err = domain.EncodeAttachmentCursor(u.kind, id, rows[len(rows)-1].ID)
			if err != nil {
				return err
			}
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return AttachmentPage{}, err
	}
	return page, nil
}
