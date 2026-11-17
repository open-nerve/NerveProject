package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// share takes the lock and the decision of a write of the caller's own row
// under the work item id, subscribeIssue's and unsubscribeIssue's (M7 P1a
// spec 2.4), and runs fn in its transaction with the work item as locked.
// It is the one exception to run's way: no place in the write gate and no
// project lock, so a subscription does not queue behind every write on the
// project's work items. First, before any lock and outside any
// transaction, the precheck: the work item read without a lock and action
// decided on it (see), which answers every refusal there, so a caller who
// would be refused takes no lock and begins no transaction. Then, in the
// transaction every write on the work-item side takes, whose lock waits
// end in 503 server_busy, the work item's row FOR SHARE (ShareIssue): it
// waits for every write of the row, and the row's deletion waits for it; a
// work item deleted while it waited is issue.not_found. Then the decision
// again on the row as locked, the guest rule first (decide), which is
// final: a project no longer visible to actor, or a work item the guest
// rule now hides from him, is issue.not_found, a role the rule no longer
// allows the Authorizer's shared.Forbidden. Without the project's lock,
// that decision does not wait for a change of actor's membership (the
// spec gives why that is safe), only for the work item's deletion. fn
// writes its rows under the work item's lock alone: every path that writes
// a work item's subscribers locks the work item's row first, the lock
// order queries/issues.sql gives at InsertSubscribers.
func (l Locks) share(ctx context.Context, actor shared.Actor, id uuid.UUID, action shared.Action, fn func(ctx context.Context, i SharedIssue) error) error {
	if _, _, err := l.see(ctx, actor, id, action); err != nil {
		return err
	}
	return l.tx.WithinTx(ctx, func(ctx context.Context) error {
		i, found, err := l.issues.ShareIssue(ctx, id)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrIssueNotFound
		}
		on := domain.Issue{ID: i.ID, WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, CreatedByID: i.CreatedByID}
		if _, err := l.decide(ctx, actor, action, i.WorkspaceID, i.ProjectID, &on, domain.ErrIssueNotFound); err != nil {
			return err
		}
		return fn(ctx, i)
	})
}
