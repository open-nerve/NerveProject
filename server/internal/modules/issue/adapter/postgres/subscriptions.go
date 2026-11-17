package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The statements of a work item's subscriptions (subscriptions.sql): its
// row FOR SHARE, the caller's row removed, and whether he subscribes.

// ShareIssue locks the undeleted work item id, archived or not, FOR SHARE,
// in the transaction ctx carries, and answers it as its row has it (M7 P1a
// spec 2.4); found is false when there is none, also when it was deleted
// while the lock waited.
func (s *Store) ShareIssue(ctx context.Context, id uuid.UUID) (app.SharedIssue, bool, error) {
	r, err := s.queries(ctx).ShareIssue(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.SharedIssue{}, false, nil
	case err != nil:
		return app.SharedIssue{}, false, fmt.Errorf("share work item %s: %w", id, err)
	}
	return app.SharedIssue{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, CreatedByID: r.CreatedByID}, true, nil
}

// RemoveSubscriber soft-deletes the work item's one undeleted row of the
// subscriber r names, at r's moment and by r's account; none for none.
func (s *Store) RemoveSubscriber(ctx context.Context, r app.SubscriberRemoval) error {
	err := s.queries(ctx).RemoveSubscriber(ctx, gen.RemoveSubscriberParams{IssueID: r.IssueID, SubscriberID: r.SubscriberID, By: r.By, At: r.At})
	if err != nil {
		return fmt.Errorf("remove subscriber %s of work item %s: %w", r.SubscriberID, r.IssueID, err)
	}
	return nil
}

// IsSubscribed reads whether the account has an undeleted row of the work
// item's subscribers (M7 P1a spec R5).
func (s *Store) IsSubscribed(ctx context.Context, issueID, account uuid.UUID) (bool, error) {
	subscribed, err := s.queries(ctx).IsSubscribed(ctx, gen.IsSubscribedParams{IssueID: issueID, SubscriberID: account})
	if err != nil {
		return false, fmt.Errorf("read whether %s subscribes to work item %s: %w", account, issueID, err)
	}
	return subscribed, nil
}
