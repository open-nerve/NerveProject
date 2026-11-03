package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

func (s *Store) LockAssetDrafts(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.queries(ctx).LockAssetDrafts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lock asset drafts: %w", err)
	}
	return rows, nil
}
func (s *Store) AssetIssueFacts(ctx context.Context, ids []uuid.UUID) ([]app.AssetParent, error) {
	rows, err := s.queries(ctx).AssetIssueFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read asset issue facts: %w", err)
	}
	out := make([]app.AssetParent, 0, len(rows))
	for _, r := range rows {
		out = append(out, app.AssetParent{ID: r.ID, DeletedAt: r.DeletedAt, DeletedByID: r.DeletedByID})
	}
	return out, nil
}
func (s *Store) AssetCommentFacts(ctx context.Context, ids []uuid.UUID) ([]app.AssetParent, error) {
	rows, err := s.queries(ctx).AssetCommentFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read asset comment facts: %w", err)
	}
	out := make([]app.AssetParent, 0, len(rows))
	for _, r := range rows {
		out = append(out, app.AssetParent{ID: r.ID, DeletedAt: r.DeletedAt, DeletedByID: r.DeletedByID})
	}
	return out, nil
}
func (s *Store) AssetDraftFacts(ctx context.Context, ids []uuid.UUID) ([]app.AssetParent, error) {
	rows, err := s.queries(ctx).AssetDraftFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read asset draft facts: %w", err)
	}
	out := make([]app.AssetParent, 0, len(rows))
	for _, r := range rows {
		out = append(out, app.AssetParent{ID: r.ID, DeletedAt: r.DeletedAt, DeletedByID: r.DeletedByID})
	}
	return out, nil
}
