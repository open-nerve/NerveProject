package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
)

// ListIntakeIssues serves GET /api/v0/projects/{project_id}/intake-issues
// (M7 P2b spec 2.3): the query as it came, every parameter and value,
// which the use case reads (domain.ParseIntakeQuery); the page as the
// use case answers it, its cursor null on the last.
func (h handler) ListIntakeIssues(ctx context.Context, req gen.ListIntakeIssuesRequestObject) (gen.ListIntakeIssuesResponseObject, error) {
	page, err := h.uc.ListIntakeIssues.Execute(ctx, req.ProjectID, rawListQuery(ctx))
	if err != nil {
		return nil, err
	}
	data := make([]gen.IntakeIssue, len(page.Data))
	for n, i := range page.Data {
		data[n] = intakeIssueRow(i)
	}
	return gen.ListIntakeIssues200JSONResponse(gen.IntakeIssuePage{Total: page.Total, Data: data, NextCursor: listCursor(page.NextCursor)}), nil
}

// GetIntakeCounts serves GET /api/v0/workspaces/{slug}/intake-counts (M7
// P2b spec 2.6): each project the use case counts, in its order.
func (h handler) GetIntakeCounts(ctx context.Context, req gen.GetIntakeCountsRequestObject) (gen.GetIntakeCountsResponseObject, error) {
	counts, err := h.uc.CountIntakeIssues.Execute(ctx, req.Slug)
	if err != nil {
		return nil, err
	}
	data := make([]gen.IntakeCount, len(counts))
	for n, c := range counts {
		data[n] = gen.IntakeCount{ProjectID: c.ProjectID, Pending: c.Pending}
	}
	return gen.GetIntakeCounts200JSONResponse(gen.IntakeCounts{Data: data}), nil
}

// intakeIssueRow is i as the intake list shows it (spec 2.3, DP-11): the
// row a project list shows (issueRow), with its intake row, which the use
// case gives every row it lists (app.ListIntakeIssues).
func intakeIssueRow(i domain.Issue) gen.IntakeIssue {
	return gen.IntakeIssue{Intake: issueIntake(*i.Intake), ID: i.ID, WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, SequenceID: i.SequenceID,
		Name: i.Name, StateID: i.StateID, Priority: apigen.IssuePriority(i.Priority), StartDate: orNull(i.StartDate), TargetDate: orNull(i.TargetDate),
		ParentID: orNull(i.ParentID), SortOrder: i.SortOrder,
		CompletedAt: orNull(i.CompletedAt), ArchivedAt: orNull(i.ArchivedAt), AssigneeIds: orEmpty(i.AssigneeIDs), LabelIds: orEmpty(i.LabelIDs),
		CycleID: orNull(i.CycleID), ModuleIds: orEmpty(i.ModuleIDs),
		SubIssuesCount: i.SubIssuesCount, LinkCount: i.LinkCount, AttachmentCount: i.AttachmentCount, CreatedByID: orNull(i.CreatedByID),
		UpdatedByID: orNull(i.UpdatedByID), CreatedAt: i.CreatedAt, UpdatedAt: i.UpdatedAt}
}
