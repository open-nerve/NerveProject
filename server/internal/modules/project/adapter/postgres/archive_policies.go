package postgresadapter

import (
	"context"
	"fmt"
	"uuid"
)

// ArchiveProjects discovers enabled, live projects in ID order. Its answer
// is only a candidate list; callers must re-read policy under the project lock.
func (s *Store) ArchiveProjects(ctx context.Context) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).ArchiveProjects(ctx)
	if err != nil {
		return nil, fmt.Errorf("read archive candidate projects: %w", err)
	}
	return ids, nil
}
