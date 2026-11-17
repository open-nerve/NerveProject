package issue

import (
	"context"
	"uuid"
)

// NotificationIssues is what the module tells the notification module about
// a project's work items (M7 P1b spec 2.6): whether any is undeleted, which
// the deletion of a deleted project's notifications waits on (2.8, R3).
// issue.Provide builds it over the pool; bootstrap hands it to the
// notification module.
type NotificationIssues interface {
	// ProjectIssuesRemain tells, without a lock, whether the project has an
	// undeleted work item, archived or not.
	ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error)
}
