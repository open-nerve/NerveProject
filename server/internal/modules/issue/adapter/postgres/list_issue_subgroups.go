package postgresadapter

import (
	"context"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Subgroups reads each nonempty cell and its independent page and lookahead.
// Original filters and group selection are separate predicates in the query.
func (s *Store) Subgroups(ctx context.Context, in app.ListRead) ([]app.ListRecord, error) {
	params, err := subgroupParameters(in)
	if err != nil {
		return nil, err
	}
	rows, err := s.queries(ctx).SubgroupIssues(ctx, params)
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
func subgroupParameters(in app.ListRead) (gen.SubgroupIssuesParams, error) {
	q := in.Query
	if !domain.ListGroupAllowed(q.GroupBy, domain.ProjectList) || !domain.ListSubGroupAllowed(q.SubGroupBy, domain.ProjectList) || q.GroupBy == q.SubGroupBy ||
		(q.GroupKey == nil) != (q.SubGroupKey == nil) || q.Cursor != nil && q.SubGroupKey == nil ||
		q.GroupKey != nil && !domain.ListGroupKeyAllowed(q.GroupBy, *q.GroupKey) || q.SubGroupKey != nil && !domain.ListGroupKeyAllowed(q.SubGroupBy, *q.SubGroupKey) {
		return gen.SubgroupIssuesParams{}, fmt.Errorf("invalid subgroup list query")
	}
	p, err := listParameters(in)
	if err != nil {
		return gen.SubgroupIssuesParams{}, err
	}
	return gen.SubgroupIssuesParams{
		GroupBy: string(q.GroupBy), GroupKey: q.GroupKey, SubGroupBy: string(q.SubGroupBy), SubGroupKey: q.SubGroupKey,
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
