package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

func (s *Store) ShareAssetWorkspaces(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.queries(ctx).ShareAssetWorkspaces(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("lock asset workspaces: %w", err)
	}
	return rows, nil
}
func (s *Store) AssetWorkspaceFacts(ctx context.Context, ids []uuid.UUID) ([]app.AssetParent, error) {
	rows, err := s.queries(ctx).AssetWorkspaceFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read asset workspace facts: %w", err)
	}
	out := make([]app.AssetParent, 0, len(rows))
	for _, r := range rows {
		out = append(out, app.AssetParent{ID: r.ID, DeletedAt: r.DeletedAt, DeletedByID: r.DeletedByID})
	}
	return out, nil
}
