package issue

import (
	"context"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// NotificationIssues is what the module tells the notification module about
// work items (M7 P1b spec 2.6): a work item's row FOR SHARE, the fan-out's
// lock, which Locks.share takes too (IssueRowLock, R8); and, without a
// lock, after it, its subscribers, its own columns and a comment's text;
// whether any of a project's work items is undeleted, which the deletion of
// a deleted project's notifications waits on (2.8, R3); and the guest rule
// the fan-out's recheck applies (2.4). Each read runs in the transaction
// ctx carries, or on the pool. issue.Provide builds it over the pool;
// bootstrap hands it to the notification module.
type NotificationIssues interface {
	IssueRowLock
	// ProjectIssuesRemain tells, without a lock, whether the project has an
	// undeleted work item, archived or not.
	ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error)
	// SubscriberIDs reads the accounts of the work item's undeleted
	// subscribers' rows, each once, in id order, none for none.
	SubscriberIDs(ctx context.Context, issueID uuid.UUID) ([]uuid.UUID, error)
	// NotificationIssue reads the undeleted work item's own columns,
	// archived or not, with its current assignees; found is false when
	// there is none.
	NotificationIssue(ctx context.Context, issueID uuid.UUID) (i NotificationIssue, found bool, err error)
	// CommentText reads the plain text the undeleted comment stores; found
	// is false when there is none.
	CommentText(ctx context.Context, commentID uuid.UUID) (text string, found bool, err error)
	// HiddenFrom is the guest rule, domain.Issue.HiddenFrom of a work item
	// whose creator is creator, nil when not known (M4 design 3.4): whether
	// it hides the work item from the account, whose grant on its project
	// is ownOnly. It reads nothing.
	HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool
}

// notificationIssues is NotificationIssues: the store's lock and reads, and
// the guest rule.
type notificationIssues struct{ *postgresadapter.Store }

func (notificationIssues) HiddenFrom(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool {
	return domain.Issue{CreatedByID: creator}.HiddenFrom(account, ownOnly)
}

// IssueRowLock is the lock NotificationIssues embeds; SharedIssue, the
// work item as the lock answers it, and NotificationIssue are the values
// it hands over: bootstrap converts them into the notification module's.
type (
	IssueRowLock      = app.IssueRowLock
	SharedIssue       = app.SharedIssue
	NotificationIssue = app.NotificationIssue
)
