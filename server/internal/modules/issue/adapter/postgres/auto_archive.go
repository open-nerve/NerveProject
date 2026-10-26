package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// AutoArchive writes system archival in one statement in the caller's
// transaction and returns only the IDs actually changed, in no defined order.
func (s *Store) AutoArchive(ctx context.Context, rows app.AutoArchiveRows) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).AutoArchive(ctx, gen.AutoArchiveParams{
		ProjectID: rows.ProjectID, StateIds: rows.StateIDs, Cutoff: rows.Cutoff, At: rows.At,
	})
	if err != nil {
		return nil, fmt.Errorf("auto-archive work items of project %s: %w", rows.ProjectID, err)
	}
	return ids, nil
}
