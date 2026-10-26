package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func (h handler) ListIssues(ctx context.Context, req gen.ListIssuesRequestObject) (gen.ListIssuesResponseObject, error) {
	page, err := h.uc.ListIssues.Execute(ctx, req.ProjectID, rawListQuery(ctx))
	if err != nil {
		return nil, err
	}
	grouped := req.Params.GroupBy != nil && req.Params.GroupKey == nil
	lanes := req.Params.SubGroupBy != nil
	return gen.ListIssues200JSONResponse(listPage(page, grouped, lanes)), nil
}

func listPage(page app.IssueListPage, grouped, lanes bool) gen.IssueListPage {
	out := gen.IssueListPage{Total: page.Total}
	if !grouped {
		data := listIssueRows(page.Data)
		out.Data, out.NextCursor = &data, listCursor(page.NextCursor)
		return out
	}
	groups := make([]gen.IssueListGroup, len(page.Groups))
	for n, g := range page.Groups {
		groups[n] = gen.IssueListGroup{Key: g.Key, Total: g.Total}
		if lanes {
			subgroups := make([]gen.IssueListSubgroup, len(g.Subgroups))
			for m, sub := range g.Subgroups {
				subgroups[m] = gen.IssueListSubgroup{Key: sub.Key, Total: sub.Total, Data: listIssueRows(sub.Data), NextCursor: listCursor(sub.NextCursor)}
			}
			groups[n].Subgroups = &subgroups
		} else {
			data := listIssueRows(g.Data)
			groups[n].Data, groups[n].NextCursor = &data, listCursor(g.NextCursor)
		}
	}
	out.Groups = &groups
	if lanes {
		totals := make([]gen.IssueListTotal, len(page.SubgroupTotals))
		for n, t := range page.SubgroupTotals {
			totals[n] = gen.IssueListTotal{Key: t.Key, Total: t.Total}
		}
		out.SubgroupTotals = &totals
	}
	return out
}
func listCursor(cursor string) nullable.Nullable[string] {
	if cursor == "" {
		return nullable.NewNullNullable[string]()
	}
	return nullable.NewNullableWithValue(cursor)
}
func listIssueRows(items []domain.Issue) []gen.Issue {
	data := make([]gen.Issue, len(items))
	for n, item := range items {
		data[n] = issueRow(item)
	}
	return data
}
