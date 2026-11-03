package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// Both fields are collected from the locked account before any asset lock.
type ProfileImageChanges struct {
	UserID                                             uuid.UUID
	AvatarCurrent, AvatarNext, CoverCurrent, CoverNext *uuid.UUID
}
type ImageBindings interface {
	LockImageBindings(context.Context, ProfileImageChanges) (time.Time, error)
	ApplyImageBindings(context.Context, ProfileImageChanges, time.Time) error
}
type ProfileImageReference struct {
	UserID                           uuid.UUID
	AvatarAssetID, CoverImageAssetID *uuid.UUID
}
type ImageURL = domain.ImageURL
type ImageReader interface {
	ReadProfileImages(context.Context, []ProfileImageReference) (map[uuid.UUID]ImageURL, error)
}
