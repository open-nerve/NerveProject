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
	DeleteIssueIntakeRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueAssigneeRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueLabelRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueCycleRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueModuleRows(ctx context.Context, d IssueDeletion) error
	DeleteIssueLinkRows(ctx context.Context, d IssueDeletion) error
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
// the rows under them, their intake rows first, whatever their status (M7
// P2a spec 2.10), the cycle and module memberships after the labels, at the
// deletion's moment and without a membership activity (M6 P3a spec 2.5).
func issueRowSteps(d IssueRowsDeleter, relations func(context.Context, IssueDeletion) error) []func(context.Context, IssueDeletion) error {
	return []func(context.Context, IssueDeletion) error{
		d.DeleteIssueRows, d.DeleteIssueIntakeRows, d.DeleteIssueAssigneeRows, d.DeleteIssueLabelRows, d.DeleteIssueCycleRows, d.DeleteIssueModuleRows, relations, d.DeleteIssueLinkRows, d.DeleteIssueSubscriberRows, d.DeleteIssueMentionRows,
		d.DeleteIssueCommentRows, d.DeleteIssueReactionRows, d.DeleteIssueCommentReactionRows, d.DeleteIssueActivityRows, d.DeleteIssueDescriptionVersionRows,
	}
}

// DeleteIssueDeps supplies lock orchestration, the shared deletion core
// and the clock a deletion's moment is read from.
type DeleteIssueDeps struct {
	Locks  Locks
	Delete *IssueRemoval
	Clock  Clock
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
// on issue.delete on it, the guest rule first: the project's admins may,
// and the member who made it (AllowCreator); then a work item in its
// project's intake refused (409 issue.in_triage, refuseTriage; M7 P2a spec
// 2.6), before any lock; then the project's write gate (503 server_busy);
// then in one transaction, the project FOR NO KEY UPDATE, the work item
// read again, the decision and the intake's refusal again, and under them
// the deletion itself (deleteLocked), its moment read from the use case's
// clock after its last lock.
func (u *DeleteIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	w := write{issue: &id, action: domain.ActionDelete, notFound: domain.ErrIssueNotFound, check: refuseTriage}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		return u.d.Delete.deleteLocked(ctx, actor.UserID, h, u.d.Clock)
	})
}
