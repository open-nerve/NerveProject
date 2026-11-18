package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// fakeIntake is the intake rows' store over the work items' fake it
// embeds: each row inserted logged with its work item and kept, in order;
// a work item read through it carries its intake row, pending, once one
// is inserted, as the GetIssue family's left join does (M7 P2a spec 2.3).
// A triage is not a submission's: the embedded writer, nil, panics.
type fakeIntake struct {
	*fakeIssues
	app.IntakeWriter
	rows []app.IntakeRow
}

func (f *fakeIntake) InsertIntake(ctx context.Context, r app.IntakeRow) error {
	if err := f.log.add(ctx, "InsertIntake", r.IssueID); err != nil {
		return err
	}
	f.rows = append(f.rows, r)
	return nil
}

func (f *fakeIntake) GetIssue(ctx context.Context, id uuid.UUID) (domain.Issue, bool, error) {
	i, found, err := f.fakeIssues.GetIssue(ctx, id)
	if found && slices.ContainsFunc(f.rows, func(r app.IntakeRow) bool { return r.IssueID == id }) {
		i.Intake = &domain.IntakeFacts{Status: domain.IntakePending}
	}
	return i, found, err
}
