package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func (s *Store) FindAsset(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	row, err := s.queries(ctx).FindAsset(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Asset{}, domain.ErrAssetNotFound
	}
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
func (s *Store) LockAssets(ctx context.Context, ids []uuid.UUID) ([]domain.Asset, error) {
	if _, ok := postgres.Tx(ctx); !ok {
		return nil, errors.New("asset locks require a transaction")
	}
	rows, err := s.queries(ctx).LockAssets(ctx, ids)
	if err != nil {
		return nil, err
	}
	return decodeAssets(rows)
}
func (s *Store) CreateAsset(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	attributes, metadata, err := encodeAsset(a)
	if err != nil {
		return domain.Asset{}, err
	}
	p := a.Placement
	row, err := s.queries(ctx).CreateAsset(ctx, gen.CreateAssetParams{ID: a.ID, UserID: p.UserID, WorkspaceID: optionalID(p.WorkspaceID), ProjectID: optionalID(p.ProjectID), IssueID: optionalID(p.IssueID), CommentID: optionalID(p.CommentID), DraftIssueID: optionalID(p.DraftID), EntityType: string(a.Purpose), ScopeKind: string(p.Scope.Kind), Attributes: attributes, Size: a.Size, IsUploaded: a.Uploaded, IsDeleted: a.DeletedAt != nil, BoundAt: a.BoundAt, UploadExpiresAt: a.UploadExpiresAt, StorageMetadata: metadata, StorageRevision: a.StorageRevision, CreatedByID: a.CreatedByID, UpdatedByID: a.UpdatedByID, DeletedByID: a.DeletedByID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt, DeletedAt: a.DeletedAt})
	if err != nil {
		return domain.Asset{}, err
	}
	return decodeAsset(row)
}
func (s *Store) ListRecovery(ctx context.Context, after uuid.UUID, limit int) ([]domain.Asset, error) {
	if limit < 1 || limit > 1000 {
		return nil, errors.New("asset recovery batch must be from 1 to 1000")
	}
	rows, err := s.queries(ctx).ListRecovery(ctx, gen.ListRecoveryParams{AfterID: after, BatchLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	return decodeAssets(rows)
}
func (s *Store) ListScope(ctx context.Context, kind domain.ScopeKind, id, after uuid.UUID, limit int) ([]domain.Asset, error) {
	if id == uuid.Nil() || (kind != domain.ScopeWorkspace && kind != domain.ScopeProject) || limit < 1 || limit > 1000 {
		return nil, domain.ErrInvalidAsset
	}
	rows, err := s.queries(ctx).ListScope(ctx, gen.ListScopeParams{ScopeKind: string(kind), ScopeID: id, AfterID: after, BatchLimit: int32(limit)})
	if err != nil {
		return nil, err
	}
	return decodeAssets(rows)
}
