package app

import (
	"context"
	"time"
	"uuid"
)

// Attachment is the metadata projection; it deliberately has no storage key or URL.
type Attachment struct {
	ID                uuid.UUID
	Name, ContentType string
	Size              int64
	CreatedAt         time.Time
	CreatedByID       *uuid.UUID
}
type AttachmentRead struct {
	Host  ContentHost
	After uuid.UUID
	Limit int
}
type AttachmentCount struct {
	IssueID uuid.UUID
	Count   int64
}
type AttachmentReader interface {
	ListAttachments(context.Context, AttachmentRead) ([]Attachment, error)
}
type AttachmentCounter interface {
	CountIssueAttachments(context.Context, []uuid.UUID) ([]AttachmentCount, error)
}
type AttachmentCountCatalog interface {
	ProjectAttachmentCounts(context.Context, []uuid.UUID, uuid.UUID, int) ([]AttachmentCount, error)
}
