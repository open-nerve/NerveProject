package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Groups reads each nonempty group's page and lookahead, in group order.
// Original filters and group selection are separate predicates in the query.
func (s *Store) Groups(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	params, err := groupParameters(in)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries(ctx).GroupIssues(ctx, params)
	if err != nil {
		return nil, err
	}
	out := make([]app.ListRecord, len(rows))
	for n, r := range rows {
		out[n], err = listRecord(gen.FlatIssuesRow(r), in.Query.Order)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
func groupParameters(in app.ListRead) (gen.GroupIssuesParams, error) {
	q := in.Query
	if (!domain.ListGroupAllowed(q.GroupBy, domain.ProjectList) && !domain.ListGroupAllowed(q.GroupBy, domain.MemberList)) || q.SubGroupBy != "" || q.SubGroupKey != nil || q.Cursor != nil && q.GroupKey == nil || q.GroupKey != nil && !domain.ListGroupKeyAllowed(q.GroupBy, *q.GroupKey) {
		return gen.GroupIssuesParams{}, fmt.Errorf("invalid grouped list query")
	}
	p, err := listParameters(in)
	if err != nil {
		return gen.GroupIssuesParams{}, err
	}
	return gen.GroupIssuesParams{
		GroupBy: string(q.GroupBy), GroupKey: q.GroupKey,
		ActorID: p.ActorID, Relation: p.Relation, RelationUserID: p.RelationUserID,
		OrderKey: p.OrderKey, Descending: p.Descending, PageSize: p.PageSize,
		HasCursor: p.HasCursor, CursorNull: p.CursorNull, CursorID: p.CursorID, CursorNum: p.CursorNum, CursorText: p.CursorText,
		Archived: p.Archived, SubIssues: p.SubIssues,
		ProjectIds: p.ProjectIds, ProjectOwn: p.ProjectOwn, StateIds: p.StateIds, StateGroups: p.StateGroups, StateKeys: p.StateKeys,
		LabelIds: p.LabelIds, LabelKeys: p.LabelKeys, AssigneeIds: p.AssigneeIds, AssigneeKeys: p.AssigneeKeys,
		FilterProjects: p.FilterProjects, FilterStates: p.FilterStates, FilterGroups: p.FilterGroups, FilterPriorities: p.FilterPriorities,
		FilterCreators: p.FilterCreators, FilterMentions: p.FilterMentions, FilterSubscribers: p.FilterSubscribers,
		FilterLabels: p.FilterLabels, LabelsNone: p.LabelsNone, FilterAssignees: p.FilterAssignees, AssigneesNone: p.AssigneesNone,
		StartNone: p.StartNone, StartFrom: p.StartFrom, StartTo: p.StartTo, TargetNone: p.TargetNone, TargetFrom: p.TargetFrom, TargetTo: p.TargetTo,
	}, nil
}
