package postgresadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) BindAttachment(ctx context.Context, in app.AttachmentBindingWrite) (domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return domain.Asset{}, errors.New("attachment binding requires a transaction")
	}
	r, err := s.queries(ctx).BindAttachment(ctx, gen.BindAttachmentParams{AssetID: in.AssetID, By: in.By, Revision: in.Revision, At: &in.At})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(r)
}
