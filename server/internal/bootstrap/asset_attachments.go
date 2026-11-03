package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
)

type issueAttachments struct {
	reader  asset.AttachmentReader
	counter asset.AttachmentCounter
	catalog asset.AttachmentCountCatalog
}

func (p issueAttachments) ListAttachments(ctx context.Context, q issue.AttachmentRead) ([]issue.Attachment, error) {
	rows, err := p.reader.ListAttachments(ctx, asset.AttachmentRead{Host: asset.ContentHost(q.Host), After: q.After, Limit: q.Limit})
	if err != nil {
		return nil, err
	}
	out := make([]issue.Attachment, len(rows))
	for i, row := range rows {
		out[i] = issue.Attachment(row)
	}
	return out, nil
}
func (p issueAttachments) CountIssueAttachments(ctx context.Context, ids []uuid.UUID) ([]issue.AttachmentCount, error) {
	rows, err := p.counter.CountIssueAttachments(ctx, ids)
	if err != nil {
		return nil, err
	}
	return issueAttachmentCounts(rows), nil
}
func (p issueAttachments) ProjectAttachmentCounts(ctx context.Context, ids []uuid.UUID, after uuid.UUID, size int) ([]issue.AttachmentCount, error) {
	rows, err := p.catalog.ProjectAttachmentCounts(ctx, ids, after, size)
	if err != nil {
		return nil, err
	}
	return issueAttachmentCounts(rows), nil
}
func issueAttachmentCounts(rows []asset.AttachmentCount) []issue.AttachmentCount {
	out := make([]issue.AttachmentCount, len(rows))
	for i, row := range rows {
		out[i] = issue.AttachmentCount(row)
	}
	return out
}
