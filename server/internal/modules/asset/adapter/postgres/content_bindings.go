package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) BindContentAsset(ctx context.Context, in app.ContentBindingWrite) (domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return domain.Asset{}, errors.New("content binding requires a transaction")
	}
	row, err := s.queries(ctx).BindContentAsset(ctx, gen.BindContentAssetParams{
		AssetID: in.AssetID, CommentID: optionalID(in.CommentID), By: in.By, Revision: in.Revision, At: &in.At,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
func (s *Store) DraftAssetIDs(ctx context.Context, draftID, afterID uuid.UUID, limit int) ([]uuid.UUID, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("draft asset directory requires a transaction")
	}
	if draftID == uuid.Nil() || limit < 1 || limit > 1000 {
		return nil, domain.ErrInvalidAsset
	}
	return s.queries(ctx).DraftAssetIDs(ctx, gen.DraftAssetIDsParams{DraftID: &draftID, AfterID: afterID, BatchLimit: int32(limit)})
}
func (s *Store) MoveDraftAsset(ctx context.Context, in app.DraftAssetWrite) (domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return domain.Asset{}, errors.New("draft asset move requires a transaction")
	}
	row, err := s.queries(ctx).MoveDraftAsset(ctx, gen.MoveDraftAssetParams{
		AssetID: in.AssetID, IssueID: optionalID(in.IssueID), ProjectID: in.ProjectID, By: in.By,
		BindAsset: in.Bind, Revision: in.Revision, At: in.At,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetScopeChanged
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
