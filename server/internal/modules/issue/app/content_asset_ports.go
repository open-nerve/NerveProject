package app

import (
	"context"
	"time"
	"uuid"
)

// ContentHost is resolved by the owning use case under its parent locks.
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
