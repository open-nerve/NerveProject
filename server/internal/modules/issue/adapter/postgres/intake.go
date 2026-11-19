package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// InsertIntake inserts the work item's intake row r, pending (M7 P2a spec
// 2.4). The use case created the work item in the same transaction, under
// the project's lock, so any violation is a bug: an internal error.
func (s *Store) InsertIntake(ctx context.Context, r app.IntakeRow) error {
	err := s.queries(ctx).InsertIntakeIssue(ctx, gen.InsertIntakeIssueParams{
		ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, By: &r.By, At: r.At,
	})
	if err != nil {
		return fmt.Errorf("insert the intake row of work item %s: %w", r.IssueID, err)
	}
	return nil
}

// TriageIntake writes the triage r of the work item's undeleted intake
// row (M7 P2a spec 2.5). The use case read the row under the project's
// lock, which every write of it holds, so one not written is an error.
func (s *Store) TriageIntake(ctx context.Context, r app.IntakeTriage) error {
	n, err := s.queries(ctx).TriageIntakeIssue(ctx, gen.TriageIntakeIssueParams{
		Status: string(r.Status), SnoozedTill: r.SnoozedTill, DuplicateToID: r.DuplicateToID, By: r.By, At: r.At, IssueID: r.IssueID,
	})
	switch {
	case err != nil:
		return fmt.Errorf("triage the intake row of work item %s: %w", r.IssueID, err)
	case n != 1:
		return fmt.Errorf("triage the intake row of work item %s: %d rows written, want 1", r.IssueID, n)
	}
	return nil
}

// DeleteIntake soft-deletes the work item's undeleted intake row alone
// (M7 P2b spec 2.5): an accepted work item's deletion from the intake. The
// use case read the row under the project's lock, which every write of it
// holds, so one not deleted is an error.
func (s *Store) DeleteIntake(ctx context.Context, d app.IntakeDeletion) error {
	n, err := s.queries(ctx).DeleteIntakeIssueRow(ctx, gen.DeleteIntakeIssueRowParams{At: d.At, By: d.By, IssueID: d.IssueID})
	switch {
	case err != nil:
		return fmt.Errorf("delete the intake row of work item %s: %w", d.IssueID, err)
	case n != 1:
		return fmt.Errorf("delete the intake row of work item %s: %d rows written, want 1", d.IssueID, n)
	}
	return nil
}
