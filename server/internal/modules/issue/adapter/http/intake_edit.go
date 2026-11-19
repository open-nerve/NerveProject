package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// UpdateIntakeIssue serves PATCH /api/v0/intake-issues/{issue_id} (M7 P2b
// spec 2.4): each field nil when not given, the target date given null
// given as none; a value given goes to the use case as it came, which
// checks it, and decides what a guest may give.
func (h handler) UpdateIntakeIssue(ctx context.Context, req gen.UpdateIntakeIssueRequestObject) (gen.UpdateIntakeIssueResponseObject, error) {
	b := req.Body
	edit := domain.IntakeIssueUpdate{DescriptionHTML: b.DescriptionHTML, SkipActivity: b.SkipActivity, Name: b.Name, Priority: (*domain.Priority)(b.Priority),
		AssigneeIDs: b.AssigneeIds, LabelIDs: b.LabelIds}
	edit.SetTargetDate, edit.TargetDate = given(b.TargetDate)
	i, err := h.uc.UpdateIntakeIssue.Execute(ctx, req.IssueID, edit)
	if err != nil {
		return nil, err
	}
	return gen.UpdateIntakeIssue200JSONResponse(issueDetail(i)), nil
}

// DeleteIntakeIssue serves DELETE /api/v0/intake-issues/{issue_id} (M7
// P2b spec 2.5).
func (h handler) DeleteIntakeIssue(ctx context.Context, req gen.DeleteIntakeIssueRequestObject) (gen.DeleteIntakeIssueResponseObject, error) {
	if err := h.uc.DeleteIntakeIssue.Execute(ctx, req.IssueID); err != nil {
		return nil, err
	}
	return gen.DeleteIntakeIssue204Response{}, nil
}
