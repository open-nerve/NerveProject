package postgresadapter

import (
	"context"
	"uuid"
)

// AssigneeIDs reads distinct accounts from undeleted assignee rows in the
// selected projects, regardless of their current membership or project role.
func (s *Store) AssigneeIDs(ctx context.Context, projectIDs []uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).ListAssigneeIDs(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, len(ids))
	copy(out, ids)
	return out, nil
}
