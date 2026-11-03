package postgresadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) SaveAsset(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	attributes, metadata, err := encodeAsset(a)
	if err != nil {
		return domain.Asset{}, err
	}
	row, err := s.queries(ctx).SaveAsset(ctx, gen.SaveAssetParams{ID: a.ID, ExpectedRevision: a.StorageRevision, Attributes: attributes, StorageMetadata: metadata, IsUploaded: a.Uploaded, IsDeleted: a.DeletedAt != nil, DeletedAt: a.DeletedAt, DeletedByID: a.DeletedByID, UpdatedAt: a.UpdatedAt, UpdatedByID: a.UpdatedByID})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
