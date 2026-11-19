package app_test

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// fakeIntake is the intake rows' store over the work items' fake it
// embeds. facts are the work items' undeleted intake rows, by work item,
// as a read of one through it carries them (Issue.Intake), as the
// GetIssue family's left join does (M7 P2a spec 2.3). Each row inserted
// is logged with its work item and kept, in order, pending; each triage
// is logged with its work item and status, kept, in order, and written
// into facts, of a work item with a row, else an error, as
// TriageIntakeIssue's statement writes one row. Every activity row
// inserted through it is kept too, in order, across inserts (written).
type fakeIntake struct {
	*fakeIssues
	rows    []app.IntakeRow
	triages []app.IntakeTriage
	facts   map[uuid.UUID]domain.IntakeFacts
	written []app.ActivityRow
}

func (f *fakeIntake) InsertIntake(ctx context.Context, r app.IntakeRow) error {
	if err := f.log.add(ctx, "InsertIntake", r.IssueID); err != nil {
		return err
	}
	f.rows = append(f.rows, r)
	if f.facts == nil {
		f.facts = map[uuid.UUID]domain.IntakeFacts{}
	}
	f.facts[r.IssueID] = domain.IntakeFacts{Status: domain.IntakePending}
	return nil
}

func (f *fakeIntake) TriageIntake(ctx context.Context, r app.IntakeTriage) error {
	if err := f.log.add(ctx, "TriageIntake", r.IssueID, r.Status); err != nil {
		return err
	}
	if _, ok := f.facts[r.IssueID]; !ok {
		return fmt.Errorf("TriageIntake: the work item %s has no intake row", r.IssueID)
	}
	f.triages = append(f.triages, r)
	f.facts[r.IssueID] = domain.IntakeFacts{Status: r.Status, SnoozedTill: r.SnoozedTill, DuplicateToID: r.DuplicateToID}
	return nil
}

func (f *fakeIntake) DeleteIntake(ctx context.Context, d app.IntakeDeletion) error {
	if err := f.log.add(ctx, "DeleteIntake", d.IssueID); err != nil {
		return err
	}
	if _, ok := f.facts[d.IssueID]; !ok {
		return fmt.Errorf("DeleteIntake: the work item %s has no intake row", d.IssueID)
	}
	delete(f.facts, d.IssueID)
	return nil
}

func (f *fakeIntake) InsertActivities(ctx context.Context, rows []app.ActivityRow) error {
	if err := f.fakeIssues.InsertActivities(ctx, rows); err != nil {
		return err
	}
	f.written = append(f.written, rows...)
	return nil
}

func (f *fakeIntake) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	i, found, err := f.fakeIssues.GetIssue(ctx, id)
	if facts, ok := f.facts[id]; found && ok {
		i.Intake = &facts
	}
	return i, found, err
}
