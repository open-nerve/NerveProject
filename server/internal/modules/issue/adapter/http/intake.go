package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// SubmitIntakeIssue serves POST /api/v0/projects/{project_id}/intake-issues
// (M7 P2a spec 2.4): the priority and the target date nil and the lists
// nil when not given, a list given empty kept empty; a value given goes to
// the use case as it came, which checks it.
func (h handler) SubmitIntakeIssue(ctx context.Context, req gen.SubmitIntakeIssueRequestObject) (gen.SubmitIntakeIssueResponseObject, error) {
	b := req.Body
	in := domain.NewIntakeIssue{DescriptionHTML: b.DescriptionHTML, Name: b.Name, Priority: (*domain.Priority)(b.Priority), TargetDate: b.TargetDate}
	if b.AssigneeIds != nil {
		in.AssigneeIDs = *b.AssigneeIds
	}
	if b.LabelIds != nil {
		in.LabelIDs = *b.LabelIds
	}
	i, err := h.uc.SubmitIntakeIssue.Execute(ctx, req.ProjectID, in)
	if err != nil {
		return nil, err
	}
	return gen.SubmitIntakeIssue201JSONResponse(issueDetail(i)), nil
}

// TriageIntakeIssue serves POST /api/v0/intake-issues/{issue_id}/triage
// (M7 P2a spec 2.5): the status as it came, any string, which the use case
// checks; the time and the work item it duplicates nil when not given.
func (h handler) TriageIntakeIssue(ctx context.Context, req gen.TriageIntakeIssueRequestObject) (gen.TriageIntakeIssueResponseObject, error) {
	t := domain.Triage{Status: domain.IntakeStatus(req.Body.Status), SnoozedTill: req.Body.SnoozedTill, DuplicateToID: req.Body.DuplicateToID}
	i, err := h.uc.TriageIntakeIssue.Execute(ctx, req.IssueID, t)
	if err != nil {
		return nil, err
	}
	return gen.TriageIntakeIssue200JSONResponse(issueDetail(i)), nil
}

// intakeOf is a work item's intake row f as its detail shows it (M7 P2a
// spec 2.8): null for none; its status, and its time and the work item it
// duplicates each null when it has none. Which of them the caller is
// answered, at which moment, is the use case's (Locks.seenOne).
func intakeOf(f *domain.IntakeFacts) nullable.Nullable[gen.IssueIntake] {
	if f == nil {
		return nullable.NewNullNullable[gen.IssueIntake]()
	}
	return nullable.NewNullableWithValue(gen.IssueIntake{Status: gen.IntakeStatus(f.Status), SnoozedTill: orNull(f.SnoozedTill),
		DuplicateToID: orNull(f.DuplicateToID)})
}
