package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type DraftAssetDeps struct {
	Locks  AssetLocker
	Reader DraftAssetReader
	Writer DraftAssetWriter
	Clock  Clock
}
type DraftAssetMover struct{ d DraftAssetDeps }

func NewDraftAssetTransfer(d DraftAssetDeps) *DraftAssetMover { return &DraftAssetMover{d: d} }
func (u *DraftAssetMover) LockDraftAssets(ctx context.Context, q DraftAssetRequest) (time.Time, error) {
	if _, err := u.checked(ctx, q); err != nil {
		return time.Time{}, err
	}
	return u.d.Clock.Now(), nil
}
func (u *DraftAssetMover) MoveDraftAssets(ctx context.Context, q DraftAssetRequest, issueID uuid.UUID, at time.Time) error {
	if at.IsZero() || q.Publish != (issueID != uuid.Nil()) {
		return domain.ErrInvalidAsset
	}
	rows, err := u.checked(ctx, q)
	if err != nil {
		return err
	}
	images := assetIDSet(q.ImageIDs)
	for _, row := range rows {
		bind := row.BoundAt == nil && (images[row.ID] || (q.Publish && row.Purpose == domain.DraftIssueAttachment))
		if !q.Publish && row.Placement.ProjectID == contentProjectID(q.NewProjectID) && !bind {
			continue
		}
		if _, err := u.d.Writer.MoveDraftAsset(ctx, DraftAssetWrite{
			AssetID: row.ID, IssueID: issueID, By: q.ActorID, ProjectID: q.NewProjectID,
			Bind: bind, Revision: row.StorageRevision, At: at,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (u *DraftAssetMover) checked(ctx context.Context, q DraftAssetRequest) ([]domain.Asset, error) {
	if !validDraftAssetRequest(q) {
		return nil, domain.ErrInvalidAsset
	}
	ids := append([]uuid.UUID(nil), q.ImageIDs...)
	after := uuid.Nil()
	for {
		page, err := u.d.Reader.DraftAssetIDs(ctx, q.DraftID, after, assetSetBatch)
		if err != nil {
			return nil, err
		}
		if len(page) > assetSetBatch {
			return nil, errors.New("draft asset directory exceeds its page limit")
		}
		for _, id := range page {
			if id.Compare(after) <= 0 {
				return nil, errors.New("draft asset directory is not strictly ascending")
			}
			after = id
		}
		ids = append(ids, page...)
		if len(page) < assetSetBatch {
			break
		}
	}
	rows, err := lockAssetSet(ctx, u.d.Locks, ids)
	if err != nil {
		return nil, err
	}
	if err := checkDraftAssetRows(q, rows); err != nil {
		return nil, err
	}
	return rows, nil
}
