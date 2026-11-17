package app

import (
	"context"
	"time"
	"uuid"
)

// DeletionStamp is a deletion's moment and deleter, which every row it
// deletes is stamped with: By is nil for nerve's own deletion, which no
// account asked for (M7 P1b spec 2.2, 2.8).
type DeletionStamp struct {
	At time.Time
	By *uuid.UUID
}

// IssueNotifications soft-deletes, in the transaction ctx carries, the
// undeleted notifications of the work item issueID of the workspace, locked
// FOR NO KEY UPDATE in id order before it writes them; a notification
// deleted before keeps its deletion (spec 2.8).
type IssueNotifications interface {
	DeleteIssueNotifications(ctx context.Context, workspaceID, issueID uuid.UUID, stamp DeletionStamp) error
}

// RowBatch is one batch of a deleted project's notifications: at most Limit
// of its undeleted ones whose ids are greater than After, in id order,
// soft-deleted at At by By.
type RowBatch struct {
	ProjectID, After, By uuid.UUID
	Limit                int
	At                   time.Time
}

// ProjectNotifications soft-deletes a batch of a deleted project's
// notifications in the transaction ctx carries, locked FOR NO KEY UPDATE in
// id order before it writes them, and answers their ids (spec 2.8).
type ProjectNotifications interface {
	DeleteProjectNotifications(ctx context.Context, b RowBatch) ([]uuid.UUID, error)
}

// ProjectIssues tells, without a lock, whether a project has an undeleted
// work item: the issue module's NotificationIssues (spec 2.6), which
// bootstrap hands over.
type ProjectIssues interface {
	ProjectIssuesRemain(ctx context.Context, projectID uuid.UUID) (bool, error)
}

// RowDeletions inserts the deletion of a deleted project's notifications in
// the publishing transaction, ctx's.
type RowDeletions interface {
	Enqueue(ctx context.Context, projectID uuid.UUID, at time.Time, by uuid.UUID) error
}
