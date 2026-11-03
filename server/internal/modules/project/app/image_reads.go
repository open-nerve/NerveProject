package app

import (
	"context"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// withProjectImages signs the final owner facts within its locked transaction.
// Built-in covers carry only their stable ID; the frontend owns their assets.
func withProjectImages(ctx context.Context, reader ImageReader, projects []domain.Project) ([]domain.Project, error) {
	refs := make([]ProjectImageReference, len(projects))
	for i, p := range projects {
		refs[i] = ProjectImageReference{WorkspaceID: p.WorkspaceID, ProjectID: p.ID, CoverImageAssetID: p.CoverImageAssetID}
	}
	images, err := reader.ReadProjectImages(ctx, refs)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(projects)
	for i := range out {
		out[i].CoverImage = nil
		if id := out[i].CoverImageAssetID; id != nil {
			if image, ok := images[*id]; ok {
				out[i].CoverImage = &image
			}
		}
	}
	return out, nil
}
