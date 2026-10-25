package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListDescriptionVersions serves GET /api/v0/issues/{issue_id}/description-versions.
func (h handler) ListDescriptionVersions(ctx context.Context, req gen.ListDescriptionVersionsRequestObject) (gen.ListDescriptionVersionsResponseObject, error) {
	page, err := h.uc.ListDescriptionVersions.Execute(ctx, req.IssueID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	data := make([]gen.DescriptionVersion, len(page.Data))
	for n, v := range page.Data {
		data[n] = versionRow(v)
	}
	var cursor *string
	if page.NextCursor != "" {
		cursor = &page.NextCursor
	}
	return gen.ListDescriptionVersions200JSONResponse{Data: data, NextCursor: orNull(cursor)}, nil
}

// GetDescriptionVersion serves GET /api/v0/issue-description-versions/{version_id}.
func (h handler) GetDescriptionVersion(ctx context.Context, req gen.GetDescriptionVersionRequestObject) (gen.GetDescriptionVersionResponseObject, error) {
	v, err := h.uc.GetDescriptionVersion.Execute(ctx, req.VersionID)
	if err != nil {
		return nil, err
	}
	return gen.GetDescriptionVersion200JSONResponse{ID: v.ID, IssueID: v.IssueID, OwnedByID: v.OwnedByID, LastSavedAt: v.LastSavedAt, CreatedAt: v.CreatedAt, DescriptionHTML: v.HTML}, nil
}
func versionRow(v domain.DescriptionVersion) gen.DescriptionVersion {
	return gen.DescriptionVersion{ID: v.ID, IssueID: v.IssueID, OwnedByID: v.OwnedByID, LastSavedAt: v.LastSavedAt, CreatedAt: v.CreatedAt}
}
