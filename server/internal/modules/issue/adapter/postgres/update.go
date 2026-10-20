package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// UpdateIssue writes the work item's fields f (M4 design 3.7). The use
// case read it under the project's lock and checked every value, so any
// violation is a bug: an internal error (500), not a domain error.
func (s *Store) UpdateIssue(ctx context.Context, f app.IssueFields) error {
	err := s.queries(ctx).UpdateIssue(ctx, gen.UpdateIssueParams{
		ID: f.ID, Name: f.Name, StateID: f.StateID, Priority: string(f.Priority), StartDate: timeOf(f.StartDate),
		TargetDate: timeOf(f.TargetDate), CompletedAt: f.CompletedAt, By: f.By, At: f.At,
	})
	if err != nil {
		return fmt.Errorf("update work item %s: %w", f.ID, err)
	}
	return nil
}

// SetArchivedAt writes the work item's archived_at, the moment or none,
// and its last writer and moment, a (M4 design 3.8).
func (s *Store) SetArchivedAt(ctx context.Context, a app.IssueArchival) error {
	err := s.queries(ctx).SetArchivedAt(ctx, gen.SetArchivedAtParams{ID: a.ID, ArchivedAt: a.ArchivedAt, By: a.By, At: a.At})
	if err != nil {
		return fmt.Errorf("archive work item %s: %w", a.ID, err)
	}
	return nil
}

// RemoveAssignees soft-deletes the work item's undeleted assignees' rows
// of the accounts r names.
func (s *Store) RemoveAssignees(ctx context.Context, r app.UnlinkRows) error {
	err := s.queries(ctx).RemoveAssignees(ctx, gen.RemoveAssigneesParams{IssueID: r.IssueID, AssigneeIds: r.Targets, By: r.By, At: r.At})
	return unlinked(err, "assignees", r.IssueID)
}

// RemoveLabels soft-deletes the work item's undeleted labels' rows of the
// labels r names.
func (s *Store) RemoveLabels(ctx context.Context, r app.UnlinkRows) error {
	err := s.queries(ctx).RemoveLabels(ctx, gen.RemoveLabelsParams{IssueID: r.IssueID, LabelIds: r.Targets, By: r.By, At: r.At})
	return unlinked(err, "labels", r.IssueID)
}

// unlinked is err, a removal of the work item issue's rows of table, as
// the removal's error; nil is nil.
func unlinked(err error, table string, issue uuid.UUID) error {
	if err != nil {
		return fmt.Errorf("remove %s of work item %s: %w", table, issue, err)
	}
	return nil
}
