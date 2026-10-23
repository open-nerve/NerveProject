package postgresadapter

import (
	"context"
	"fmt"
	"uuid"
)

// Ancestry reads the work item id and those above it, each once, in no
// order (M4 design 3.9): its parent, its parent's parent and so on.
func (s *Store) Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).Ancestry(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the work items above work item %s: %w", id, err)
	}
	return ids, nil
}
