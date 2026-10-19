package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The statements of a deleted work item (app.IssueRowsDeleter,
// delete_issue.sql): each soft-deletes the work item's undeleted rows of
// one table.

// deletion is d as the statements take it; their parameters are of one
// shape.
func deletion(d app.IssueDeletion) gen.DeleteIssueRowParams {
	return gen.DeleteIssueRowParams{At: d.At, DeletedBy: d.By, IssueID: d.IssueID}
}

// deletedRows is err, the deletion of the work item d's rows of what, as
// the deletion's error; nil is nil.
func deletedRows(err error, what string, d app.IssueDeletion) error {
	if err != nil {
		return fmt.Errorf("delete %s of work item %s: %w", what, d.IssueID, err)
	}
	return nil
}

// DeleteIssueRow soft-deletes the work item, archived or not.
func (s *Store) DeleteIssueRow(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueRow(ctx, deletion(d)), "the row", d)
}

// DeleteIssueAssigneeRows soft-deletes the work item's assignees' rows.
func (s *Store) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueAssigneeRows(ctx, gen.DeleteIssueAssigneeRowsParams(deletion(d))), "the assignees", d)
}

// DeleteIssueLabelRows soft-deletes the rows of the labels on the work
// item.
func (s *Store) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueLabelRows(ctx, gen.DeleteIssueLabelRowsParams(deletion(d))), "the labels", d)
}

// DeleteIssueSubscriberRows soft-deletes the work item's subscribers'
// rows.
func (s *Store) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueSubscriberRows(ctx, gen.DeleteIssueSubscriberRowsParams(deletion(d))), "the subscribers", d)
}

// DeleteIssueActivityRows soft-deletes the work item's activity.
func (s *Store) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueActivityRows(ctx, gen.DeleteIssueActivityRowsParams(deletion(d))), "the activity", d)
}
