package app_test

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// fakeIssues as deleteIssue reads and writes the work items
// (app.IssueRowsDeleter): the subtree it is told, and each deletion's
// arguments kept, each call logged.

// Subtree logs the read and answers subtree, or the work item id alone
// when subtree is nil.
func (f *fakeIssues) Subtree(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Subtree", id); err != nil {
		return nil, err
	}
	if f.subtree == nil {
		return []uuid.UUID{id}, nil
	}
	return f.subtree, nil
}

// deleteRows logs a deletion of name's rows, d's work items', and keeps d,
// or answers the failure.
func (f *fakeIssues) deleteRows(ctx context.Context, name string, d app.IssueDeletion) error {
	if err := f.log.add(ctx, name, d.IssueIDs); err != nil {
		return err
	}
	f.deletions = append(f.deletions, d)
	return nil
}

func (f *fakeIssues) DeleteIssueRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueRows", d)
}

func (f *fakeIssues) DeleteIssueAssigneeRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueAssigneeRows", d)
}

func (f *fakeIssues) DeleteIssueLabelRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueLabelRows", d)
}

func (f *fakeIssues) DeleteIssueRelationRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueRelationRows", d)
}

func (f *fakeIssues) DeleteIssueSubscriberRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueSubscriberRows", d)
}

func (f *fakeIssues) DeleteIssueActivityRows(ctx context.Context, d app.IssueDeletion) error {
	return f.deleteRows(ctx, "DeleteIssueActivityRows", d)
}
