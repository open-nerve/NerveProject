package postgresadapter

import (
	"context"
	"uuid"
)

// ShareImageWorkspaces reuses the same ordered, undeleted-owner share query.
func (s *Store) ShareImageWorkspaces(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	return s.ShareAssetWorkspaces(ctx, ids)
}
