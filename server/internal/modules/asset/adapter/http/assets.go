package httpadapter

import (
	"context"
	"maps"
	"slices"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (h handler) CreateAsset(ctx context.Context, r gen.CreateAssetRequestObject) (gen.CreateAssetResponseObject, error) {
	in := r.Body
	if in.Scope.Kind == gen.AssetScopeKindUser && in.Scope.ID != nil {
		return nil, shared.Invalid(shared.FieldError{Field: "scope.id", Code: shared.FieldNotAllowed, Message: "must be omitted for the current user"})
	}
	scope := domain.Scope{Kind: domain.ScopeKind(in.Scope.Kind)}
	if in.Scope.ID != nil {
		scope.ID = *in.Scope.ID
	}
	got, err := h.uc.Prepare.Execute(ctx, domain.NewAsset{Purpose: domain.Purpose(in.Purpose), Scope: scope, Name: in.Name, Size: in.Size, ContentType: in.ContentType})
	if err != nil {
		return nil, err
	}
	headers := make([]gen.UploadHeader, 0, len(got.Upload.Headers))
	for _, name := range slices.Sorted(maps.Keys(got.Upload.Headers)) {
		headers = append(headers, gen.UploadHeader{Name: name, Value: got.Upload.Headers[name]})
	}
	return gen.CreateAsset201JSONResponse{Asset: assetDTO(got.Asset), Upload: gen.Upload{Method: gen.UploadMethod(got.Upload.Method), URL: got.Upload.URL, Headers: headers, ExpiresAt: got.Upload.ExpiresAt}}, nil
}
func (h handler) GetAsset(ctx context.Context, r gen.GetAssetRequestObject) (gen.GetAssetResponseObject, error) {
	got, err := h.uc.Get.Execute(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return gen.GetAsset200JSONResponse(assetDTO(got)), nil
}
func (h handler) ResolveAssetContent(ctx context.Context, r gen.ResolveAssetContentRequestObject) (gen.ResolveAssetContentResponseObject, error) {
	got, err := h.uc.ResolveContent.Execute(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return gen.ResolveAssetContent200JSONResponse{URL: got.URL, ExpiresAt: got.ExpiresAt}, nil
}
func assetDTO(a domain.Asset) gen.Asset {
	scope := gen.AssetScope{Kind: gen.AssetScopeKind(a.Placement.Scope.Kind)}
	if a.Placement.Scope.ID != uuid.Nil() {
		scope.ID = &a.Placement.Scope.ID
	}
	state := gen.AssetStatePending
	if a.Uploaded {
		state = gen.AssetStateUploaded
	}
	width, height := nullable.NewNullNullable[int](), nullable.NewNullNullable[int]()
	if a.Width != nil {
		width = nullable.NewNullableWithValue(*a.Width)
	}
	if a.Height != nil {
		height = nullable.NewNullableWithValue(*a.Height)
	}
	return gen.Asset{ID: a.ID, Purpose: gen.AssetPurpose(a.Purpose), Scope: scope, Name: a.Name, Size: a.Size, ContentType: a.ContentType,
		State: state, IsBound: a.BoundAt != nil, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, Width: width, Height: height}
}
