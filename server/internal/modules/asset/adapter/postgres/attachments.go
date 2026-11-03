package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func attachmentReadTransaction(ctx context.Context) error {
	if _, ok := postgres.Tx(ctx); !ok {
		return errors.New("attachment read requires the caller's transaction")
	}
	return nil
}
func (s *Store) ListAttachments(ctx context.Context, q app.AttachmentRead) ([]app.Attachment, error) {
	if err := attachmentReadTransaction(ctx); err != nil {
		return nil, err
	}
	h := q.Host
	if h.WorkspaceID == uuid.Nil() || q.Limit < 1 || q.Limit > 101 {
		return nil, domain.ErrInvalidAsset
	}
	var kind domain.ScopeKind
	var purpose domain.Purpose
	switch {
	case h.DraftID != uuid.Nil() && h.IssueID == uuid.Nil() && h.CommentID == uuid.Nil():
		kind, purpose = domain.ScopeDraft, domain.DraftIssueAttachment
	case h.DraftID == uuid.Nil() && h.IssueID != uuid.Nil() && h.ProjectID != uuid.Nil():
		kind, purpose = domain.ScopeIssue, domain.IssueAttachment
		if h.CommentID != uuid.Nil() {
			kind, purpose = domain.ScopeComment, domain.CommentAttachment
		}
	default:
		return nil, domain.ErrInvalidAsset
	}
	rows, err := s.queries(ctx).ListAttachments(ctx, gen.ListAttachmentsParams{WorkspaceID: h.WorkspaceID, ProjectID: optionalID(h.ProjectID), IssueID: optionalID(h.IssueID), CommentID: optionalID(h.CommentID), DraftID: optionalID(h.DraftID), ScopeKind: string(kind), Purpose: string(purpose), AfterID: q.After, PageLimit: int32(q.Limit)})
	if err != nil {
		return nil, err
	}
	out := make([]app.Attachment, 0, len(rows))
	for _, row := range rows {
		var attrs assetAttributes
		if err := json.Unmarshal(row.Attributes, &attrs); err != nil {
			return nil, err
		}
		out = append(out, app.Attachment{ID: row.ID, Name: attrs.Name, ContentType: attrs.ContentType, Size: row.Size, CreatedAt: row.CreatedAt.UTC(), CreatedByID: row.CreatedByID})
	}
	return out, nil
}
func (s *Store) AttachmentCounts(ctx context.Context, ids []uuid.UUID) ([]app.AttachmentCount, error) {
	if err := attachmentReadTransaction(ctx); err != nil {
		return nil, err
	}
	if len(ids) < 1 || len(ids) > 1000 {
		return nil, domain.ErrInvalidAsset
	}
	rows, err := s.queries(ctx).AttachmentCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]app.AttachmentCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, app.AttachmentCount{IssueID: requiredID(row.IssueID), Count: row.AttachmentCount})
	}
	return out, nil
}
func (s *Store) ProjectAttachmentCounts(ctx context.Context, ids []uuid.UUID, after uuid.UUID, size int) ([]app.AttachmentCount, error) {
	if err := attachmentReadTransaction(ctx); err != nil {
		return nil, err
	}
	if len(ids) < 1 || len(ids) > 1000 || size != 1000 {
		return nil, domain.ErrInvalidAsset
	}
	rows, err := s.queries(ctx).ProjectAttachmentCounts(ctx, gen.ProjectAttachmentCountsParams{ProjectIds: ids, AfterID: after, PageLimit: int32(size)})
	if err != nil {
		return nil, err
	}
	out := make([]app.AttachmentCount, 0, len(rows))
	for _, row := range rows {
		out = append(out, app.AttachmentCount{IssueID: requiredID(row.IssueID), Count: row.AttachmentCount})
	}
	return out, nil
}
