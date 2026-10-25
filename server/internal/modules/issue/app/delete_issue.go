package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IssueDeletion is a work item's deletion as its rows take it (M4 design
// 3.6, 3.9): the work items it deletes, the one asked for and each work
// item under it, however deep, archived ones too, its subtree; and the
// moment and the account that every row it deletes is stamped with.
type IssueDeletion struct {
	IssueIDs []uuid.UUID
	By       uuid.UUID
	At       time.Time
}

// IssueRowsDeleter reads a deleted work item's subtree and soft-deletes
// its rows, one table a call, in the transaction ctx carries, under its
// project's lock (M4 design 3.5, 3.6): each Delete method soft-deletes the
// undeleted rows of its table of the work items d deletes, at d.At by d.By,
// and leaves a row deleted before as it is.
type IssueRowsDeleter interface {
	// Subtree reads the undeleted work item and each undeleted work item
	// under it, however deep, archived ones too, in id order.
	Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error)
	DeleteIssueRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueAssigneeRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueLabelRows(ctx context.Context, d IssueDeletion) error
	// DeleteIssueRelationRows soft-deletes the relations that name one of
	// the work items at either end, its other end of any project, locked
	// in id order first (convention 5's exception).
	DeleteIssueRelationRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueSubscriberRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueMentionRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueDescriptionVersionRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueCommentRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueReactionRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueCommentReactionRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueActivityRows(ctx context.Context, d IssueDeletion) error
}

// issueRowSteps are d's statements of a work item's deletion in the global
// order of M4 design 3.5, as a project's (rowSteps): the work items, then
// the rows under them.
func issueRowSteps(d IssueRowsDeleter) []func(context.Context, IssueDeletion) error {
	return []func(context.Context, IssueDeletion) error{
		d.DeleteIssueRows, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueRelationRows, d.DeleteIssueSubscriberRows, d.DeleteIssueMentionRows,
		d.DeleteIssueCommentRows, d.DeleteIssueReactionRows, d.DeleteIssueCommentReactionRows, d.DeleteIssueActivityRows, d.DeleteIssueDescriptionVersionRows,
	}
}

// DeleteIssueDeps are what deleteIssue writes through.
type DeleteIssueDeps struct {
	Locks Locks
	Rows  IssueRowsDeleter
	Clock Clock
}

// DeleteIssue deletes a work item: DELETE /api/v0/issues/{issue_id} (M4
// design 3.5, 3.6, 5.1).
type DeleteIssue struct {
	d DeleteIssueDeps
}

// NewDeleteIssue returns the use case.
func NewDeleteIssue(d DeleteIssueDeps) *DeleteIssue {
	return &DeleteIssue{d: d}
}

// Execute runs, through Locks, the deletion of the work item id (M4 design
// 3.5, 3.6, 3.9, 11.11): its precheck, the work item read and the decision
// on issue.delete on it, the guest rule first, before any lock: the
// project's admins may, and the member who made it (AllowCreator); then
// the project's write gate (503 server_busy); then in one transaction, the
// project FOR NO KEY UPDATE, the work item read again and the decision
// again, and under them: an archived project refuses it (409
// project.archived); its subtree, the work item and each one under it,
// however deep, archived ones too, read under the lock, which every write
// of a parent holds; the clock; the subtree's work items and each row
// under them soft-deleted, table by table in the global order
// (issueRowSteps), all of it the caller's at that moment. An archived work
// item is deleted as any other (3.8). A failing statement ends it with its
// failure, and none after it runs.
func (u *DeleteIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	w := write{issue: &id, action: domain.ActionDelete, notFound: domain.ErrIssueNotFound}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		subtree, err := u.d.Rows.Subtree(ctx, id)
		if err != nil {
			return err
		}
		d := IssueDeletion{IssueIDs: subtree, By: actor.UserID, At: u.d.Clock.Now()}
		for _, step := range issueRowSteps(u.d.Rows) {
			if err := step(ctx, d); err != nil {
				return err
			}
		}
		return nil
	})
}
