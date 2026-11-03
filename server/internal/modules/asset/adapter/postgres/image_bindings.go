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

func (s *Store) BindImage(ctx context.Context, in app.ImageBindingWrite) (domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return domain.Asset{}, errors.New("image binding requires a transaction")
	}
	row, err := s.queries(ctx).BindImage(ctx, gen.BindImageParams{
		At: &in.At, By: in.By, ProjectID: optionalID(in.ProjectID),
		AssetID: in.AssetID, Revision: in.Revision,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
