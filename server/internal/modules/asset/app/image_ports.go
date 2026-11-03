package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ImageBindingHost struct {
	ApplicantID            uuid.UUID
	WorkspaceID, ProjectID uuid.UUID
	NewProject             bool
}
type ImageChange struct {
	Field             string
	Purpose           domain.Purpose
	CurrentID, NextID *uuid.UUID
}
type ImageBindingRequest struct {
	Host    ImageBindingHost
	Changes []ImageChange
}
type ImageBindings interface {
	LockImageBindings(context.Context, ImageBindingRequest) (time.Time, error)
	ApplyImageBindings(context.Context, ImageBindingRequest, time.Time) error
}

// ImageBindingWrite changes no applicant, object metadata or existing parent.
// Only a new_project preparation gains its newly inserted project parent.
type ImageBindingWrite struct {
	AssetID, ProjectID, By uuid.UUID
	Revision               int64
	At                     time.Time
}
type ImageBindingWriter interface {
	BindImage(context.Context, ImageBindingWrite) (domain.Asset, error)
}
