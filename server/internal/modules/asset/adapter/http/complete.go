package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
)

func (h handler) CompleteAsset(ctx context.Context, r gen.CompleteAssetRequestObject) (gen.CompleteAssetResponseObject, error) {
	got, err := h.uc.Complete.Execute(ctx, r.ID)
	if err != nil {
		return nil, err
	}
	return gen.CompleteAsset200JSONResponse(assetDTO(got)), nil
}
