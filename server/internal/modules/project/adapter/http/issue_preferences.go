package httpadapter

import (
	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// filterAtoms distinguishes an omitted key from an explicit empty array.
func filterAtoms(v *[]string) []string {
	if v == nil {
		return nil
	}
	return *v
}
func filterArray(v []string) *[]string {
	if v == nil {
		return nil
	}
	return &v
}
func grouping(v nullable.Nullable[string]) *string {
	if v.IsNull() {
		return nil
	}
	s := v.MustGet()
	return &s
}

func issuePreferencesIn(b *gen.ProjectPreferencesUpdate, p *domain.PreferencesPatch) {
	if f := b.Filters; f != nil {
		p.Filters = &domain.Filters{StateID: filterAtoms(f.StateID), StateGroup: filterAtoms(f.StateGroup), Priority: filterAtoms(f.Priority),
			LabelID: filterAtoms(f.LabelID), AssigneeID: filterAtoms(f.AssigneeID), CreatedByID: filterAtoms(f.CreatedByID), MentionID: filterAtoms(f.MentionID),
			SubscriberID: filterAtoms(f.SubscriberID), StartDate: filterAtoms(f.StartDate), TargetDate: filterAtoms(f.TargetDate)}
	}
	if d := b.DisplayFilters; d != nil {
		p.DisplayFilters = &domain.DisplayFilters{Layout: d.Layout, GroupBy: grouping(d.GroupBy), SubGroupBy: grouping(d.SubGroupBy), OrderBy: d.OrderBy,
			SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: domain.DisplayCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
	}
	if d := b.DisplayProperties; d != nil {
		p.DisplayProperties = &domain.DisplayProperties{Assignee: d.Assignee, StartDate: d.StartDate, DueDate: d.DueDate, Labels: d.Labels, Key: d.Key,
			Priority: d.Priority, State: d.State, SubIssueCount: d.SubIssueCount, Link: d.Link, CreatedOn: d.CreatedOn, UpdatedOn: d.UpdatedOn}
	}
}

func issuePreferencesOut(p domain.Preferences, out *gen.ProjectPreferences) {
	f, d, v := p.Filters, p.DisplayFilters, p.DisplayProperties
	out.Filters = gen.ProjectIssueFilters{StateID: filterArray(f.StateID), StateGroup: filterArray(f.StateGroup), Priority: filterArray(f.Priority),
		LabelID: filterArray(f.LabelID), AssigneeID: filterArray(f.AssigneeID), CreatedByID: filterArray(f.CreatedByID), MentionID: filterArray(f.MentionID),
		SubscriberID: filterArray(f.SubscriberID), StartDate: filterArray(f.StartDate), TargetDate: filterArray(f.TargetDate)}
	out.DisplayFilters = gen.ProjectDisplayFilters{Layout: d.Layout, GroupBy: orNull(d.GroupBy), SubGroupBy: orNull(d.SubGroupBy), OrderBy: d.OrderBy,
		SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: gen.ProjectListCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
	out.DisplayProperties = gen.ProjectDisplayProperties{Assignee: v.Assignee, StartDate: v.StartDate, DueDate: v.DueDate, Labels: v.Labels, Key: v.Key,
		Priority: v.Priority, State: v.State, SubIssueCount: v.SubIssueCount, Link: v.Link, CreatedOn: v.CreatedOn, UpdatedOn: v.UpdatedOn}
}
