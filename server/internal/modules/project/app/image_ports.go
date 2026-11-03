package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type CoverImageChange struct {
	ActorID, WorkspaceID, ProjectID uuid.UUID
	NewProject                      bool
	Current, Next                   *uuid.UUID
}

type ImageBindings interface {
	LockImageBindings(context.Context, CoverImageChange) (time.Time, error)
	ApplyImageBindings(context.Context, CoverImageChange, time.Time) error
}

type CallerLock interface {
	LockCaller(context.Context, shared.Actor, time.Time) error
}

// ImageGate shares the composition's project gate with issue and asset.
// Enter happens before opening the image transaction, after visibility checks.
type ImageGate interface {
	Enter(context.Context, ...uuid.UUID) (func(), error)
}

type ProjectImageReference struct {
	WorkspaceID, ProjectID uuid.UUID
	CoverImageAssetID      *uuid.UUID
}
type ImageURL = domain.ImageURL
type ImageReader interface {
	ReadProjectImages(context.Context, []ProjectImageReference) (map[uuid.UUID]ImageURL, error)
}
