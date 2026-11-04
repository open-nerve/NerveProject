package httpadapter

import (
	"context"
	"net/http"
	"net/url"
	"uuid"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ListDraftPreparationsUseCase interface {
	Execute(context.Context, uuid.UUID, *int, *string) (app.DraftPreparationPage, error)
}

func (h handler) ListDraftAssetPreparations(ctx context.Context, r gen.ListDraftAssetPreparationsRequestObject) (gen.ListDraftAssetPreparationsResponseObject, error) {
	page, err := h.uc.ListDraftPreparations.Execute(ctx, r.DraftID, r.Params.Limit, r.Params.Cursor)
	if err != nil {
		return nil, err
	}
	out := gen.ListDraftAssetPreparations200JSONResponse{Data: make([]gen.Asset, len(page.Data)), NextCursor: nullable.NewNullNullable[apigen.NextCursor]()}
	if page.NextCursor != nil {
		out.NextCursor = nullable.NewNullableWithValue(*page.NextCursor)
	}
	for i, a := range page.Data {
		out.Data[i] = assetDTO(a)
	}
	return out, nil
}

func withPreparationQuery(next gen.StrictHandlerFunc, operationID string) gen.StrictHandlerFunc {
	if operationID != "ListDraftAssetPreparations" {
		return next
	}
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		q, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			return nil, shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The request query could not be decoded.")
		}
		for key, values := range q {
			if (key != "limit" && key != "cursor") || len(values) != 1 {
				return nil, shared.NewError(shared.KindBadRequest, shared.CodeBadRequest, "The preparation query has an unknown or repeated parameter.")
			}
		}
		return next(ctx, w, r, request)
	}
}
