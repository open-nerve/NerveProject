package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AttachmentBindingWrite struct {
	AssetID, By uuid.UUID
	Revision    int64
	At          time.Time
}

// AttachmentWriter binds a published ordinary attachment in the caller's
// transaction. It cannot change its purpose or parent chain.
type AttachmentWriter interface {
	BindAttachment(context.Context, AttachmentBindingWrite) (domain.Asset, error)
}

type AttachmentPublication struct {
	Writer AttachmentWriter
	Events shared.EventPublisher
}

func (p AttachmentPublication) Bind(ctx context.Context, row domain.Asset, by uuid.UUID, at time.Time) (domain.Asset, error) {
	if row.BoundAt != nil || !row.IsOrdinaryAttachment() {
		return row, nil
	}
	bound, err := p.Writer.BindAttachment(ctx, AttachmentBindingWrite{AssetID: row.ID, By: by, Revision: row.StorageRevision, At: at})
	if err != nil {
		return domain.Asset{}, err
	}
	if err := publishAttachmentChange(ctx, p.Events, bound, by, at, true); err != nil {
		return domain.Asset{}, err
	}
	return bound, nil
}

func publishAttachmentChange(ctx context.Context, events shared.EventPublisher, row domain.Asset, by uuid.UUID, at time.Time, added bool) error {
	if row.Purpose != domain.IssueAttachment || !row.IsOrdinaryAttachment() || row.BoundAt == nil {
		return nil
	}
	kind := domain.EventAttachmentDeleted
	change := shared.Change{Field: "issue", OldID: &row.Placement.IssueID}
	if added {
		kind = domain.EventAttachmentAdded
		change.OldID, change.NewID = nil, &row.Placement.IssueID
	}
	return events.Publish(ctx, shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: at,
		WorkspaceID: row.Placement.WorkspaceID, ProjectID: row.Placement.ProjectID, ActorID: &by,
		Entity: shared.EntityRef{Kind: "asset", ID: row.ID}, Changes: []shared.Change{change}})
}

// A competing complete may have published and automatically bound this same
// ordinary attachment. That is the winning result of the operation, not a
// content edit or a move. All other placement and binding changes still fail.
func completionPlacement(before, row domain.Asset, host Host) error {
	if before.BoundAt == nil && row.BoundAt != nil && row.IsOrdinaryAttachment() && isPublished(row) {
		before.BoundAt = row.BoundAt
	}
	return currentPlacement(before, row, host)
}
