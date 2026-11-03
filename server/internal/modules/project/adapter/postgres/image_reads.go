package postgresadapter

import (
	"context"
	"uuid"
)

// ShareImageProjects uses the existing ordered, undeleted-project SHARE query.
// The owner has already locked its caller and workspace in this transaction.
func (s *Store) ShareImageProjects(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	return s.ShareIssueProjects(ctx, ids)
}
