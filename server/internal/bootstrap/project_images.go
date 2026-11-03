package bootstrap

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type projectImageBindings struct{ bindings asset.ImageBindings }

func projectImageRequest(q project.CoverImageChange) asset.ImageBindingRequest {
	return asset.ImageBindingRequest{Host: asset.ImageBindingHost{ApplicantID: q.ActorID, WorkspaceID: q.WorkspaceID, ProjectID: q.ProjectID, NewProject: q.NewProject}, Changes: []asset.ImageChange{
		{Field: "cover_image_asset_id", Purpose: assetdomain.ProjectCover, CurrentID: q.Current, NextID: q.Next},
	}}
}
func (p projectImageBindings) LockImageBindings(ctx context.Context, q project.CoverImageChange) (time.Time, error) {
	return p.bindings.LockImageBindings(ctx, projectImageRequest(q))
}
func (p projectImageBindings) ApplyImageBindings(ctx context.Context, q project.CoverImageChange, at time.Time) error {
	return p.bindings.ApplyImageBindings(ctx, projectImageRequest(q), at)
}

type projectImageReader struct{ images asset.ImageReader }

func (p projectImageReader) ReadProjectImages(ctx context.Context, refs []project.ProjectImageReference) (map[uuid.UUID]project.ImageURL, error) {
	var query []asset.ImageReference
	for _, ref := range refs {
		if ref.CoverImageAssetID != nil {
			query = append(query, asset.ImageReference{AssetID: *ref.CoverImageAssetID, Purpose: assetdomain.ProjectCover, WorkspaceID: ref.WorkspaceID, ProjectID: ref.ProjectID})
		}
	}
	images, err := p.images.ReadBoundImages(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]project.ImageURL, len(images))
	for id, image := range images {
		out[id] = project.ImageURL{URL: image.URL, ExpiresAt: image.ExpiresAt}
	}
	return out, nil
}

type projectImageGate struct{ gate writeGate }

func (g projectImageGate) Enter(ctx context.Context, ids ...uuid.UUID) (func(), error) {
	if _, nested := postgres.Tx(ctx); nested {
		return nil, errors.New("project image operation cannot join an existing transaction")
	}
	return g.gate.Enter(ctx, ids...)
}
