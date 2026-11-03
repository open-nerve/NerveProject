package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ContentBindingDeps struct {
	Locks  AssetLocker
	Writer ContentBindingWriter
	Clock  Clock
}
type ContentBinder struct{ d ContentBindingDeps }

func NewContentBindings(d ContentBindingDeps) *ContentBinder { return &ContentBinder{d: d} }
func (u *ContentBinder) LockContent(ctx context.Context, q ContentBindRequest) (time.Time, error) {
	if _, err := u.checked(ctx, q); err != nil {
		return time.Time{}, err
	}
	return u.d.Clock.Now(), nil
}
func (u *ContentBinder) BindContent(ctx context.Context, q ContentBindRequest, at time.Time) error {
	if at.IsZero() {
		return domain.ErrInvalidAsset
	}
	rows, err := u.checked(ctx, q)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.BoundAt != nil {
			continue
		}
		if _, err := u.d.Writer.BindContentAsset(ctx, ContentBindingWrite{
			AssetID: row.ID, CommentID: q.Host.CommentID, By: q.ActorID, Revision: row.StorageRevision, At: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (u *ContentBinder) checked(ctx context.Context, q ContentBindRequest) ([]domain.Asset, error) {
	if q.ActorID == uuid.Nil() || q.Host.WorkspaceID == uuid.Nil() || q.Host.ProjectID == uuid.Nil() ||
		q.Host.IssueID == uuid.Nil() || q.Host.DraftID != uuid.Nil() || (q.NewComment && q.Host.CommentID == uuid.Nil()) {
		return nil, domain.ErrInvalidAsset
	}
	ids := append(append([]uuid.UUID(nil), q.ImageIDs...), q.AttachmentIDs...)
	rows, err := lockAssetSet(ctx, u.d.Locks, ids)
	if err != nil {
		return nil, err
	}
	if err := checkContentRows(q, rows); err != nil {
		return nil, err
	}
	return rows, nil
}
