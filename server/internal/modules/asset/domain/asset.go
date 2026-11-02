package domain

import (
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	ErrInvalidAsset         = errors.New("invalid asset")
	ErrAssetTooLarge        = errors.New("asset too large")
	ErrAssetTypeUnsupported = errors.New("unsupported asset type")
	ErrInvalidMetadata      = errors.New("invalid asset storage metadata")
	ErrAssetContentInvalid  = errors.New("invalid asset content")
	ErrAssetNotFound        = shared.NewError(shared.KindNotFound, "asset.not_found", "The asset does not exist, or you cannot see it.")
	ErrAssetScopeChanged    = shared.NewError(shared.KindConflict, "asset.scope_changed", "The asset's scope or storage state changed. Please try again.")
	ErrAssetNotUploaded     = shared.NewError(shared.KindConflict, "asset.not_uploaded", "The asset's content has not been completed.")
)

// These two categories are mapped to HTTP at the adapter boundary.
const (
	CodeAssetTooLarge        = "asset.too_large"
	CodeAssetTypeUnsupported = "asset.type_unsupported"
)

type NewAsset struct {
	Purpose     Purpose
	Scope       Scope
	Name        string
	Size        int64
	ContentType string
}

// Asset separates a verified object from its later binding to a business host.
type Asset struct {
	ID                                    uuid.UUID
	Purpose                               Purpose
	Placement                             Placement
	Name, ContentType                     string
	Size                                  int64
	Width, Height                         *int
	Uploaded                              bool
	CreatedAt, UpdatedAt                  time.Time
	CreatedByID, UpdatedByID, DeletedByID *uuid.UUID
	DeletedAt, BoundAt, UploadExpiresAt   *time.Time
	StorageRevision                       int64
	Metadata                              StorageMetadata
}

// ContentProof describes all bytes and, for image purposes, only the decoded header.
type ContentProof struct {
	SHA256        string
	Width, Height *int
}

func ValidateNewAsset(a NewAsset, limit int64) error {
	if limit < 1 || limit > MaxObjectSize || len(NewAssetFields(a)) != 0 {
		return ErrInvalidAsset
	}
	if a.Size > limit || a.Size > MaxObjectSize {
		return ErrAssetTooLarge
	}
	if a.Purpose.ImageRequired() && !RasterMediaType(a.ContentType) {
		return ErrAssetTypeUnsupported
	}
	return nil
}

// NewAssetFields gives semantic errors in the public input's field order.
func NewAssetFields(a NewAsset) []shared.FieldError {
	var fields []shared.FieldError
	add := func(field, code, message string) {
		fields = append(fields, shared.FieldError{Field: field, Code: code, Message: message})
	}
	if !a.Purpose.Valid() {
		add("purpose", shared.FieldNotAllowed, "is not a supported purpose")
	} else if !a.Scope.ValidFor(a.Purpose) {
		add("scope", shared.FieldNotAllowed, "does not identify a host allowed for this purpose")
	}
	if !validName(a.Name) {
		add("name", shared.FieldInvalidFormat, "must have 1 to 255 characters without path separators or control characters")
	}
	if a.Size < 1 {
		add("size", shared.FieldOutOfRange, "must be a positive byte length")
	}
	if !ValidMediaType(a.ContentType) {
		add("content_type", shared.FieldInvalidFormat, "must be a canonical MIME type without parameters")
	}
	return fields
}

// RasterMediaType is the allowlist for image validation and inline transfers.
func RasterMediaType(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
