package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (h handler) DuplicateAsset(ctx context.Context, r gen.DuplicateAssetRequestObject) (gen.DuplicateAssetResponseObject, error) {
	in := r.Body
	if in.Scope.Kind == gen.AssetScopeKindUser && in.Scope.ID != nil {
		return nil, shared.Invalid(shared.FieldError{Field: "scope.id", Code: shared.FieldNotAllowed, Message: "must be omitted for the current user"})
	}
	scope := domain.Scope{Kind: domain.ScopeKind(in.Scope.Kind)}
	if in.Scope.ID != nil {
		scope.ID = *in.Scope.ID
	}
	got, err := h.uc.Duplicate.Execute(ctx, r.ID, domain.DuplicateAsset{Purpose: domain.Purpose(in.Purpose), Scope: scope, Name: in.Name})
	if err != nil {
		return nil, err
	}
	return gen.DuplicateAsset201JSONResponse(assetDTO(got)), nil
}
func (h handler) DeleteAsset(ctx context.Context, r gen.DeleteAssetRequestObject) (gen.DeleteAssetResponseObject, error) {
	if err := h.uc.Delete.Execute(ctx, r.ID); err != nil {
		return nil, err
	}
	return gen.DeleteAsset204Response{}, nil
}
