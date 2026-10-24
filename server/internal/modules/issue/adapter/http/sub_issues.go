package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListSubIssues serves GET /api/v0/issues/{issue_id}/sub-issues.
func (h handler) ListSubIssues(ctx context.Context, req gen.ListSubIssuesRequestObject) (gen.ListSubIssuesResponseObject, error) {
	items, err := h.uc.ListSubIssues.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ListSubIssues200JSONResponse(issueCollection(items)), nil
}

// AddSubIssues serves POST /api/v0/issues/{issue_id}/sub-issues: the work
// items go to the use case as they came, which checks them.
func (h handler) AddSubIssues(ctx context.Context, req gen.AddSubIssuesRequestObject) (gen.AddSubIssuesResponseObject, error) {
	items, err := h.uc.AddSubIssues.Execute(ctx, req.IssueID, req.Body.SubIssueIds)
	if err != nil {
		return nil, err
	}
	return gen.AddSubIssues200JSONResponse(issueCollection(items)), nil
}

// issueCollection lists items in order, using [] for none. Each explicit Issue
// mapping omits description_html, which only the detail contract exposes.
func issueCollection(items []domain.Issue) gen.IssueCollection {
	data := make([]gen.Issue, len(items))
	for n, i := range items {
		data[n] = issueRow(i)
	}
	return gen.IssueCollection{Data: data}
}
