package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

func (s *Store) AssetProjectArchived(ctx context.Context, id uuid.UUID) (bool, bool, error) {
	archived, err := s.queries(ctx).AssetProjectArchived(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, fmt.Errorf("read asset project state: %w", err)
	}
	return archived, true, nil
}
func (s *Store) AssetProjectFacts(ctx context.Context, ids []uuid.UUID) ([]app.AssetParent, error) {
	rows, err := s.queries(ctx).AssetProjectFacts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read asset project facts: %w", err)
	}
	out := make([]app.AssetParent, 0, len(rows))
	for _, r := range rows {
		out = append(out, app.AssetParent{ID: r.ID, DeletedAt: r.DeletedAt, DeletedByID: r.DeletedByID})
	}
	return out, nil
}
