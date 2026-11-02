package postgresadapter

import (
	"bytes"
	"encoding/json"
	"io"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type assetAttributes struct {
	Name        string `json:"name"`
	ContentType string `json:"content_type"`
	Width       *int   `json:"width,omitempty"`
	Height      *int   `json:"height,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
}

func encodeAsset(a domain.Asset) ([]byte, []byte, error) {
	if a.ID == uuid.Nil() || a.CreatedAt.IsZero() || a.UpdatedAt.IsZero() || a.StorageRevision < 0 {
		return nil, nil, domain.ErrInvalidAsset
	}
	if err := a.Placement.Validate(a.Purpose); err != nil {
		return nil, nil, err
	}
	if err := domain.ValidateNewAsset(domain.NewAsset{Purpose: a.Purpose, Scope: a.Placement.Scope, Name: a.Name, Size: a.Size, ContentType: a.ContentType}, domain.MaxObjectSize); err != nil {
		return nil, nil, err
	}
	if err := a.Metadata.Validate(a.ID); err != nil {
		return nil, nil, err
	}
	attrs := assetAttributes{Name: a.Name, ContentType: a.ContentType, Width: a.Width, Height: a.Height}
	if a.Metadata.Final != nil {
		attrs.SHA256 = a.Metadata.Final.SHA256
	}
	attributes, err := json.Marshal(attrs)
	if err != nil {
		return nil, nil, err
	}
	metadata, err := json.Marshal(a.Metadata)
	return attributes, metadata, err
}
func decodeAsset(row gen.FileAsset) (domain.Asset, error) {
	var attrs assetAttributes
	decoder := json.NewDecoder(bytes.NewReader(row.Attributes))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&attrs) != nil || decoder.Decode(new(any)) != io.EOF {
		return domain.Asset{}, domain.ErrInvalidAsset
	}
	metadata, err := domain.DecodeStorageMetadata(row.ID, row.StorageMetadata)
	if err != nil {
		return domain.Asset{}, err
	}
	p := domain.Placement{Scope: domain.Scope{Kind: domain.ScopeKind(row.ScopeKind)}, UserID: row.UserID, WorkspaceID: requiredID(row.WorkspaceID), ProjectID: requiredID(row.ProjectID), IssueID: requiredID(row.IssueID), CommentID: requiredID(row.CommentID), DraftID: requiredID(row.DraftIssueID)}
	switch p.Scope.Kind {
	case domain.ScopeWorkspace, domain.ScopeNewProject:
		p.Scope.ID = p.WorkspaceID
	case domain.ScopeProject:
		p.Scope.ID = p.ProjectID
	case domain.ScopeIssue, domain.ScopeNewComment:
		p.Scope.ID = p.IssueID
	case domain.ScopeComment:
		p.Scope.ID = p.CommentID
	case domain.ScopeDraft:
		p.Scope.ID = p.DraftID
	}
	a := domain.Asset{ID: row.ID, Purpose: domain.Purpose(row.EntityType), Placement: p, Name: attrs.Name, ContentType: attrs.ContentType, Size: row.Size, Width: attrs.Width, Height: attrs.Height, Uploaded: row.IsUploaded, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC(), CreatedByID: row.CreatedByID, UpdatedByID: row.UpdatedByID, DeletedByID: row.DeletedByID, DeletedAt: utcTime(row.DeletedAt), BoundAt: utcTime(row.BoundAt), UploadExpiresAt: utcTime(row.UploadExpiresAt), StorageRevision: row.StorageRevision, Metadata: metadata}
	if err := p.Validate(a.Purpose); err != nil {
		return domain.Asset{}, err
	}
	return a, nil
}
func decodeAssets(rows []gen.FileAsset) ([]domain.Asset, error) {
	out := make([]domain.Asset, 0, len(rows))
	for _, row := range rows {
		a, err := decodeAsset(row)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}
func optionalID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil() {
		return nil
	}
	return &id
}
func requiredID(id *uuid.UUID) uuid.UUID {
	if id == nil {
		return uuid.Nil()
	}
	return *id
}
func utcTime(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	utc := at.UTC()
	return &utc
}
