package domain

import (
	"errors"
	"time"
	"uuid"
)

var (
	ErrInvalidAsset         = errors.New("invalid asset")
	ErrAssetTooLarge        = errors.New("asset too large")
	ErrAssetTypeUnsupported = errors.New("unsupported asset type")
	ErrInvalidMetadata      = errors.New("invalid asset storage metadata")
	ErrAssetContentInvalid  = errors.New("invalid asset content")
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
	if limit < 1 || limit > MaxObjectSize || a.Size < 1 || !validName(a.Name) || !ValidMediaType(a.ContentType) || !a.Scope.ValidFor(a.Purpose) {
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

// RasterMediaType is the allowlist for image validation and inline transfers.
func RasterMediaType(value string) bool {
	switch value {
	case "image/jpeg", "image/png", "image/webp", "image/gif":
		return true
	default:
		return false
	}
}
