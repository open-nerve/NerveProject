package postgresadapter

import (
	"fmt"
	"strings"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func flatParameters(in app.ListRead) (gen.FlatIssuesParams, error) {
	q := in.Query
	if q.GroupBy != "" || q.SubGroupBy != "" || q.GroupKey != nil || q.SubGroupKey != nil {
		return gen.FlatIssuesParams{}, fmt.Errorf("invalid flat list query")
	}
	return listParameters(in)
}

// listParameters maps the shared filters, catalogs and exact cursor once.
// Allocate even empty arrays; append each directory's parallel fields together.
func listParameters(in app.ListRead) (gen.FlatIssuesParams, error) {
	q, f, d := in.Query, in.Query.Filters, in.Directories
	validRelation := q.Relation == "assigned" || q.Relation == "created" || q.Relation == "subscribed"
	if (in.TargetMemberID == nil && q.Relation != "") || (in.TargetMemberID != nil && !validRelation) {
		return gen.FlatIssuesParams{}, fmt.Errorf("list relation and target member do not match")
	}
	if !domain.ListOrderAllowed(q.Order) || q.Limit < 1 || q.Limit > 100 {
		return gen.FlatIssuesParams{}, fmt.Errorf("invalid list query")
	}
	p := gen.FlatIssuesParams{
		ActorID: in.ActorID, Relation: q.Relation, RelationUserID: in.TargetMemberID,
		OrderKey: strings.TrimPrefix(string(q.Order), "-"), Descending: strings.HasPrefix(string(q.Order), "-"),
		PageSize: int32(q.Limit + 1), Archived: f.Archived, SubIssues: f.SubIssues,
		FilterProjects: listArray(f.ProjectIDs), FilterStates: listArray(f.StateIDs),
		FilterCreators: listArray(f.CreatorIDs), FilterMentions: listArray(f.MentionIDs), FilterSubscribers: listArray(f.SubscriberIDs),
		FilterLabels: listArray(f.Labels.IDs), LabelsNone: f.Labels.None,
		FilterAssignees: listArray(f.Assignees.IDs), AssigneesNone: f.Assignees.None,
		FilterGroups: make([]string, 0, len(f.StateGroups)), FilterPriorities: make([]string, 0, len(f.Priorities)),
		ProjectIds: make([]uuid.UUID, 0, len(d.Projects)), ProjectOwn: make([]bool, 0, len(d.Projects)),
		StateIds: make([]uuid.UUID, 0, len(d.States)), StateGroups: make([]string, 0, len(d.States)), StateKeys: make([]string, 0, len(d.States)),
		LabelIds: make([]uuid.UUID, 0, len(d.Labels)), LabelKeys: make([]string, 0, len(d.Labels)),
		AssigneeIds: make([]uuid.UUID, 0, len(d.Assignees)), AssigneeKeys: make([]string, 0, len(d.Assignees)),
	}
	for _, v := range d.Projects {
		p.ProjectIds = append(p.ProjectIds, v.ID)
		p.ProjectOwn = append(p.ProjectOwn, v.OwnOnly)
	}
	for _, v := range d.States {
		p.StateIds = append(p.StateIds, v.ID)
		p.StateGroups = append(p.StateGroups, string(v.Group))
		p.StateKeys = append(p.StateKeys, v.Key)
	}
	for _, v := range d.Labels {
		p.LabelIds = append(p.LabelIds, v.ID)
		p.LabelKeys = append(p.LabelKeys, v.Key)
	}
	for _, v := range d.Assignees {
		p.AssigneeIds = append(p.AssigneeIds, v.ID)
		p.AssigneeKeys = append(p.AssigneeKeys, v.Key)
	}
	for _, v := range f.StateGroups {
		p.FilterGroups = append(p.FilterGroups, string(v))
	}
	for _, v := range f.Priorities {
		p.FilterPriorities = append(p.FilterPriorities, string(v))
	}
	p.StartNone, p.StartFrom, p.StartTo = dateParameters(f.Start)
	p.TargetNone, p.TargetFrom, p.TargetTo = dateParameters(f.Target)
	if q.Cursor != nil {
		if err := cursorParameters(&p, *q.Cursor, q.Order); err != nil {
			return p, err
		}
	}
	return p, nil
}

func listArray[T any](values []T) []T {
	out := make([]T, len(values))
	copy(out, values)
	return out
}

// Empty endpoint text represents an open bound; SQL NULLIF converts it
// before date comparison. Dates remain pure dates, with no duration math.
func dateParameters(filters []domain.DateFilter) ([]bool, []string, []string) {
	none, from, to := make([]bool, len(filters)), make([]string, len(filters)), make([]string, len(filters))
	for n, f := range filters {
		none[n] = f.None
		if f.From != nil {
			from[n] = f.From.String()
		}
		if f.To != nil {
			to[n] = f.To.String()
		}
	}
	return none, from, to
}
