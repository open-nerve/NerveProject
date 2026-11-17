package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
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

// SubscriberIDs reads, without a lock, the accounts of the work item
// issueID's undeleted subscribers' rows, each once, in id order, none for
// none (notification_issues.sql), in the transaction ctx carries or on the
// pool.
func (s *Store) SubscriberIDs(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).SubscriberIDs(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("read the subscribers of work item %s: %w", issueID, err)
	}
	return ids, nil
}

// NotificationIssue reads, without a lock, the undeleted work item id's
// own columns, archived or not, with its current assignees
// (notification_issues.sql), in the transaction ctx carries or on the
// pool; found is false when there is none.
func (s *Store) NotificationIssue(ctx context.Context, id uuid.UUID) (app.NotificationIssue, bool, error) {
	r, err := s.queries(ctx).NotificationIssue(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.NotificationIssue{}, false, nil
	case err != nil:
		return app.NotificationIssue{}, false, fmt.Errorf("read work item %s for its notifications: %w", id, err)
	}
	return app.NotificationIssue{Name: r.Name, SequenceID: int(r.SequenceID), StateID: r.StateID, CreatedByID: r.CreatedByID,
		AssigneeIDs: r.AssigneeIds}, true, nil
}

// CommentText reads, without a lock, the plain text the undeleted comment
// id stores (notification_issues.sql), in the transaction ctx carries or
// on the pool; found is false when there is none.
func (s *Store) CommentText(ctx context.Context, id uuid.UUID) (string, bool, error) {
	text, err := s.queries(ctx).CommentText(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return "", false, nil
	case err != nil:
		return "", false, fmt.Errorf("read the text of comment %s: %w", id, err)
	}
	return text, true, nil
}
