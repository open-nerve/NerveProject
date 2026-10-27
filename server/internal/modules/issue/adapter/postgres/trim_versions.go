package postgresadapter

import (
	"context"
	"fmt"
	"uuid"
)

// TrimDescriptionVersions deletes old live versions in the caller's transaction.
func (s *Store) TrimDescriptionVersions(ctx context.Context, id uuid.UUID) (int, error) {
	n, err := s.queries(ctx).TrimDescriptionVersions(ctx, id)
	if err != nil {
		return 0, fmt.Errorf("trim description versions of project %s: %w", id, err)
	}
	return int(n), nil
}
