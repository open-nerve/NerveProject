package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
)

// DeleteIssueNotifications soft-deletes the undeleted notifications of a
// deleted work item at the stamp, in the transaction ctx carries
// (app.IssueNotifications, deletion.sql).
func (s *Store) DeleteIssueNotifications(ctx context.Context, workspaceID, issueID uuid.UUID, stamp app.DeletionStamp) error {
	if stamp.At.IsZero() || (stamp.By != nil && *stamp.By == (uuid.UUID{})) {
		return errors.New("invalid notification deletion stamp")
	}
	err := s.queries(ctx).DeleteIssueNotifications(ctx, gen.DeleteIssueNotificationsParams{At: stamp.At, DeletedBy: stamp.By, WorkspaceID: workspaceID, IssueID: issueID})
	if err != nil {
		return fmt.Errorf("delete the notifications of work item %s: %w", issueID, err)
	}
	return nil
}

// DeleteProjectNotifications soft-deletes a batch of a deleted project's
// notifications in the transaction ctx carries and answers their ids
// (app.ProjectNotifications, deletion.sql).
func (s *Store) DeleteProjectNotifications(ctx context.Context, b app.RowBatch) ([]uuid.UUID, error) {
	if b.Limit < 1 || b.Limit > math.MaxInt32 {
		return nil, fmt.Errorf("notification deletion batch must be from 1 to 2147483647, got %d", b.Limit)
	}
	ids, err := s.queries(ctx).DeleteProjectNotifications(ctx, gen.DeleteProjectNotificationsParams{At: b.At, DeletedBy: b.By, ProjectID: b.ProjectID, After: b.After, Batch: int32(b.Limit)})
	if err != nil {
		return nil, fmt.Errorf("delete a batch of the project's notifications: %w", err)
	}
	return ids, nil
}
