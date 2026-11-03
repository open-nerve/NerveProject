package bootstrap

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

type profileImageBindings struct{ bindings asset.ImageBindings }

func profileImageRequest(q identity.ProfileImageChanges) asset.ImageBindingRequest {
	return asset.ImageBindingRequest{Host: asset.ImageBindingHost{ApplicantID: q.UserID}, Changes: []asset.ImageChange{
		{Field: "avatar_asset_id", Purpose: assetdomain.UserAvatar, CurrentID: q.AvatarCurrent, NextID: q.AvatarNext},
		{Field: "cover_image_asset_id", Purpose: assetdomain.UserCover, CurrentID: q.CoverCurrent, NextID: q.CoverNext},
	}}
}
func (p profileImageBindings) LockImageBindings(ctx context.Context, q identity.ProfileImageChanges) (time.Time, error) {
	return p.bindings.LockImageBindings(ctx, profileImageRequest(q))
}
func (p profileImageBindings) ApplyImageBindings(ctx context.Context, q identity.ProfileImageChanges, at time.Time) error {
	return p.bindings.ApplyImageBindings(ctx, profileImageRequest(q), at)
}

type profileImageReader struct{ images asset.ImageReader }

func (p profileImageReader) ReadProfileImages(ctx context.Context, refs []identity.ProfileImageReference) (map[uuid.UUID]identity.ImageURL, error) {
	var query []asset.ImageReference
	for _, ref := range refs {
		if ref.AvatarAssetID != nil {
			query = append(query, asset.ImageReference{AssetID: *ref.AvatarAssetID, Purpose: assetdomain.UserAvatar, UserID: ref.UserID})
		}
		if ref.CoverImageAssetID != nil {
			query = append(query, asset.ImageReference{AssetID: *ref.CoverImageAssetID, Purpose: assetdomain.UserCover, UserID: ref.UserID})
		}
	}
	images, err := p.images.ReadBoundImages(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]identity.ImageURL, len(images))
	for id, image := range images {
		out[id] = identity.ImageURL{URL: image.URL, ExpiresAt: image.ExpiresAt}
	}
	return out, nil
}

type workspaceImageBindings struct{ bindings asset.ImageBindings }

func workspaceImageRequest(q workspace.LogoImageChange) asset.ImageBindingRequest {
	return asset.ImageBindingRequest{Host: asset.ImageBindingHost{ApplicantID: q.ActorID, WorkspaceID: q.WorkspaceID}, Changes: []asset.ImageChange{
		{Field: "logo_asset_id", Purpose: assetdomain.WorkspaceLogo, CurrentID: q.Current, NextID: q.Next},
	}}
}
func (p workspaceImageBindings) LockImageBindings(ctx context.Context, q workspace.LogoImageChange) (time.Time, error) {
	return p.bindings.LockImageBindings(ctx, workspaceImageRequest(q))
}
func (p workspaceImageBindings) ApplyImageBindings(ctx context.Context, q workspace.LogoImageChange, at time.Time) error {
	return p.bindings.ApplyImageBindings(ctx, workspaceImageRequest(q), at)
}

type workspaceImageReader struct{ images asset.ImageReader }

func (p workspaceImageReader) ReadWorkspaceImages(ctx context.Context, refs []workspace.WorkspaceImageReference) (map[uuid.UUID]workspace.ImageURL, error) {
	var query []asset.ImageReference
	for _, ref := range refs {
		if ref.LogoAssetID != nil {
			query = append(query, asset.ImageReference{AssetID: *ref.LogoAssetID, Purpose: assetdomain.WorkspaceLogo, WorkspaceID: ref.WorkspaceID})
		}
	}
	images, err := p.images.ReadBoundImages(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]workspace.ImageURL, len(images))
	for id, image := range images {
		out[id] = workspace.ImageURL{URL: image.URL, ExpiresAt: image.ExpiresAt}
	}
	return out, nil
}

type memberAvatarReader struct{ images asset.ImageReader }

func (p memberAvatarReader) ReadMemberAvatars(ctx context.Context, refs []workspace.MemberAvatarReference) (map[uuid.UUID]workspace.ImageURL, error) {
	var query []asset.ImageReference
	for _, ref := range refs {
		if ref.AssetID != nil {
			query = append(query, asset.ImageReference{AssetID: *ref.AssetID, Purpose: assetdomain.UserAvatar, UserID: ref.UserID})
		}
	}
	images, err := p.images.ReadBoundImages(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]workspace.ImageURL, len(images))
	for id, image := range images {
		out[id] = workspace.ImageURL{URL: image.URL, ExpiresAt: image.ExpiresAt}
	}
	return out, nil
}
