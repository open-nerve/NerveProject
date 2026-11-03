package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AttachmentEventStore writes the activity and issue stamp in the existing
// transaction under the publisher's project write lock; it never opens one.
type AttachmentEventStore interface {
	RecordAttachmentChange(context.Context, ActivityRow) error
}

func NewAttachmentChanges(rows AttachmentEventStore) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "issue_attachment_changes", Types: []string{"asset.attachment_added", "asset.attachment_deleted"},
		Handle: func(ctx context.Context, event shared.Event) error {
			id, added, err := attachmentIssue(event)
			if err != nil {
				return err
			}
			return rows.RecordAttachmentChange(ctx, ActivityRow{ID: uuid.NewV7(), WorkspaceID: event.WorkspaceID,
				ProjectID: event.ProjectID, IssueID: id, ActorID: event.ActorID, At: event.OccurredAt,
				Activity: domain.AttachmentActivity(event.Entity.ID, added)})
		}}
}

func attachmentIssue(e shared.Event) (uuid.UUID, bool, error) {
	invalid := errors.New("invalid attachment change event")
	if e.ID == uuid.Nil() || e.Entity.Kind != "asset" || e.Entity.ID == uuid.Nil() || e.WorkspaceID == uuid.Nil() ||
		e.ProjectID == uuid.Nil() || e.ActorID == nil || *e.ActorID == uuid.Nil() || e.OccurredAt.IsZero() || len(e.Changes) != 1 {
		return uuid.Nil(), false, invalid
	}
	c := e.Changes[0]
	if c.Field != "issue" || c.Old != nil || c.New != nil {
		return uuid.Nil(), false, invalid
	}
	switch e.Type {
	case "asset.attachment_added":
		if c.NewID != nil && *c.NewID != uuid.Nil() && c.OldID == nil {
			return *c.NewID, true, nil
		}
	case "asset.attachment_deleted":
		if c.OldID != nil && *c.OldID != uuid.Nil() && c.NewID == nil {
			return *c.OldID, false, nil
		}
	}
	return uuid.Nil(), false, invalid
}
