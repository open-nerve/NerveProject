package postgresadapter

import (
	"context"
	"fmt"
	"uuid"
)

// ProjectIssuesRemain tells, without a lock, whether the project has an
// undeleted work item, archived or not (notification_issues.sql), in the
// transaction ctx carries or on the pool.
func (s *Store) ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error) {
	remain, err := s.queries(ctx).ProjectIssuesRemain(ctx, projectID)
	if err != nil {
		return false, fmt.Errorf("read whether project %s has undeleted work items: %w", projectID, err)
	}
	return remain, nil
}
