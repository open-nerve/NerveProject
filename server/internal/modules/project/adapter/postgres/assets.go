package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// AssetProject reads the undeleted project id: whether it is archived and
// whether its intake is turned on, in one statement, without a lock.
func (s *Store) AssetProject(ctx context.Context, id uuid.UUID) (app.AssetProject, bool, error) {
	r, err := s.queries(ctx).AssetProject(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return app.AssetProject{}, false, nil
	}
	if err != nil {
		return app.AssetProject{}, false, fmt.Errorf("read asset project state: %w", err)
	}
	return app.AssetProject{Archived: r.Archived, IntakeView: r.IntakeView}, true, nil
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
