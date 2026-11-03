package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// The caller holds its credential/account lock in this same transaction.
func withProfileImages(ctx context.Context, reader ImageReader, u domain.User) (domain.User, error) {
	images, err := reader.ReadProfileImages(ctx, []ProfileImageReference{{UserID: u.ID, AvatarAssetID: u.AvatarAssetID, CoverImageAssetID: u.CoverImageAssetID}})
	if err != nil {
		return domain.User{}, err
	}
	find := func(id *uuid.UUID) *domain.ImageURL {
		if id != nil {
			if image, ok := images[*id]; ok {
				return &image
			}
		}
		return nil
	}
	u.Avatar = find(u.AvatarAssetID)
	u.CoverImage = find(u.CoverImageAssetID)
	return u, nil
}
