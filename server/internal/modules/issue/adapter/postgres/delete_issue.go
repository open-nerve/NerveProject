package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The statements of a deleted work item (app.IssueRowsDeleter,
// delete_issue.sql): its subtree, and the statements that soft-delete the
// subtree's undeleted rows of one table each.

// Subtree reads the undeleted work item id and each undeleted work item
// under it, however deep, archived ones too, in id order (M4 design 3.6,
// 3.9).
func (s *Store) Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).IssueSubtree(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("read the work items under work item %s: %w", id, err)
	}
	return ids, nil
}

// deletion is d as the statements take it; their parameters are of one
// shape.
func deletion(d app.IssueDeletion) gen.DeleteIssueRowsParams {
	return gen.DeleteIssueRowsParams{At: d.At, DeletedBy: d.By, IssueIds: d.IssueIDs}
}

// deletedRows is err, the deletion of the work items d's rows of what, as
// the deletion's error; nil is nil.
func deletedRows(err error, what string, d app.IssueDeletion) error {
	if err != nil {
		return fmt.Errorf("delete %s of work items %v: %w", what, d.IssueIDs, err)
	}
	return nil
}

// DeleteIssueRows soft-deletes the work items, archived or not.
func (s *Store) DeleteIssueRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueRows(ctx, deletion(d)), "the rows", d)
}

// DeleteIssueAssigneeRows soft-deletes the work items' assignees' rows.
func (s *Store) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueAssigneeRows(ctx, gen.DeleteIssueAssigneeRowsParams(deletion(d))), "the assignees", d)
}

// DeleteIssueLabelRows soft-deletes the rows of the labels on the work
// items.
func (s *Store) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueLabelRows(ctx, gen.DeleteIssueLabelRowsParams(deletion(d))), "the labels", d)
}

// DeleteIssueSubscriberRows soft-deletes the work items' subscribers'
// rows.
func (s *Store) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueSubscriberRows(ctx, gen.DeleteIssueSubscriberRowsParams(deletion(d))), "the subscribers", d)
}

// DeleteIssueActivityRows soft-deletes the work items' activity.
func (s *Store) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return deletedRows(s.queries(ctx).DeleteIssueActivityRows(ctx, gen.DeleteIssueActivityRowsParams(deletion(d))), "the activity", d)
}
