package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// LogoImageChange is the owner-verified current and final reference.
type LogoImageChange struct {
	ActorID, WorkspaceID uuid.UUID
	Current, Next        *uuid.UUID
}

// ImageBindings runs inside the owner's transaction after its authorization
// and workspace lock. Both steps use the same request and last-lock time.
type ImageBindings interface {
	LockImageBindings(context.Context, LogoImageChange) (time.Time, error)
	ApplyImageBindings(context.Context, LogoImageChange, time.Time) error
}

type WorkspaceImageReference struct {
	WorkspaceID uuid.UUID
	LogoAssetID *uuid.UUID
}

type ImageURL = domain.ImageURL

// ImageReader signs only the fixed references read under owner locks.
type ImageReader interface {
	ReadWorkspaceImages(context.Context, []WorkspaceImageReference) (map[uuid.UUID]ImageURL, error)
}

// ImageWorkspaceSharer locks the undeleted candidates FOR SHARE in UUID order
// and returns only the IDs it locked. It never discovers additional owners.
type ImageWorkspaceSharer interface {
	ShareImageWorkspaces(context.Context, []uuid.UUID) ([]uuid.UUID, error)
}

// MemberAvatarReference comes only from a workspace-authorized public profile.
type MemberAvatarReference struct {
	UserID  uuid.UUID
	AssetID *uuid.UUID
}
type MemberAvatarReader interface {
	ReadMemberAvatars(context.Context, []MemberAvatarReference) (map[uuid.UUID]ImageURL, error)
}
