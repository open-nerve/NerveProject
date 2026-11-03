package bootstrap

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
)

type issueContentBindings struct{ bindings asset.ContentBinding }

func contentAssetRequest(q issue.ContentBindRequest) asset.ContentBindRequest {
	return asset.ContentBindRequest{ActorID: q.ActorID, Host: asset.ContentHost(q.Host),
		ImageIDs: q.ImageIDs, AttachmentIDs: q.AttachmentIDs, NewComment: q.NewComment}
}
func (p issueContentBindings) LockContent(ctx context.Context, q issue.ContentBindRequest) (time.Time, error) {
	return p.bindings.LockContent(ctx, contentAssetRequest(q))
}
func (p issueContentBindings) BindContent(ctx context.Context, q issue.ContentBindRequest, at time.Time) error {
	return p.bindings.BindContent(ctx, contentAssetRequest(q), at)
}

type issueDraftAssetTransfer struct{ assets asset.DraftAssetTransfer }

func (p issueDraftAssetTransfer) LockDraftAssets(ctx context.Context, q issue.DraftAssetRequest) (time.Time, error) {
	return p.assets.LockDraftAssets(ctx, asset.DraftAssetRequest(q))
}
func (p issueDraftAssetTransfer) MoveDraftAssets(ctx context.Context, q issue.DraftAssetRequest, id uuid.UUID, at time.Time) error {
	return p.assets.MoveDraftAssets(ctx, asset.DraftAssetRequest(q), id, at)
}
