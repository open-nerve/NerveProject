package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ContentHost struct{ WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID }
type ContentBindRequest struct {
	ActorID                 uuid.UUID
	Host                    ContentHost
	ImageIDs, AttachmentIDs []uuid.UUID
	NewComment              bool
}
type ContentBinding interface {
	LockContent(context.Context, ContentBindRequest) (time.Time, error)
	BindContent(context.Context, ContentBindRequest, time.Time) error
}
type ContentBindingWrite struct {
	AssetID, CommentID, By uuid.UUID
	Revision               int64
	At                     time.Time
}
type ContentBindingWriter interface {
	BindContentAsset(context.Context, ContentBindingWrite) (domain.Asset, error)
}

type DraftAssetRequest struct {
	ActorID, WorkspaceID, DraftID uuid.UUID
	OldProjectID, NewProjectID    *uuid.UUID
	ImageIDs                      []uuid.UUID
	Publish                       bool
}
type DraftAssetTransfer interface {
	LockDraftAssets(context.Context, DraftAssetRequest) (time.Time, error)
	MoveDraftAssets(context.Context, DraftAssetRequest, uuid.UUID, time.Time) error
}
type DraftAssetReader interface {
	DraftAssetIDs(context.Context, uuid.UUID, uuid.UUID, int) ([]uuid.UUID, error)
}
type DraftAssetWrite struct {
	AssetID, IssueID, By uuid.UUID
	ProjectID            *uuid.UUID
	Bind                 bool
	Revision             int64
	At                   time.Time
}
type DraftAssetWriter interface {
	MoveDraftAsset(context.Context, DraftAssetWrite) (domain.Asset, error)
}
