package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

func withWorkspaceImages(ctx context.Context, reader ImageReader, list []domain.Workspace) ([]domain.Workspace, error) {
	refs := make([]WorkspaceImageReference, len(list))
	for i, w := range list {
		refs[i] = WorkspaceImageReference{WorkspaceID: w.ID, LogoAssetID: w.LogoAssetID}
	}
	images, err := reader.ReadWorkspaceImages(ctx, refs)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(list)
	for i := range out {
		out[i].Logo = nil
		if id := out[i].LogoAssetID; id != nil {
			if image, ok := images[*id]; ok {
				out[i].Logo = &image
			}
		}
	}
	return out, nil
}

func imageWorkspaceIDs(list []domain.Workspace) []uuid.UUID {
	ids := make([]uuid.UUID, len(list))
	for i, w := range list {
		ids[i] = w.ID
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return slices.Compact(ids)
}
