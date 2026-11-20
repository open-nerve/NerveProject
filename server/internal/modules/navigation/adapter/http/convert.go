package httpadapter

import (
	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newView is b as the use case takes it, unchecked (P3b spec 2.4, DP-22):
// a description, an icon and filters left out are their zero values, none;
// display settings left out are nil, which the use case fills with the
// defaults.
func newView(b gen.ViewCreate) domain.NewView {
	n := domain.NewView{Name: b.Name, DisplayFilters: displayFiltersIn(b.DisplayFilters), DisplayProperties: displayPropertiesIn(b.DisplayProperties)}
	if b.Description != nil {
		n.Description = *b.Description
	}
	if b.LogoProps != nil {
		n.LogoProps = logoIn(*b.LogoProps)
	}
	if b.Filters != nil {
		n.Filters = filtersIn(*b.Filters)
	}
	return n
}

// patchOf is b as the use case takes it, unchecked (spec 2.4): each field
// it gives, whole; each it leaves out nil, which keeps the view's. Within
// the filters, a key given empty stays apart from one left out.
func patchOf(b gen.ViewUpdate) domain.Patch {
	p := domain.Patch{Name: b.Name, Description: b.Description, DisplayFilters: displayFiltersIn(b.DisplayFilters),
		DisplayProperties: displayPropertiesIn(b.DisplayProperties)}
	if b.LogoProps != nil {
		logo := logoIn(*b.LogoProps)
		p.LogoProps = &logo
	}
	if b.Filters != nil {
		f := filtersIn(*b.Filters)
		p.Filters = &f
	}
	return p
}

// viewOf is v as the API shows it: project_id null for a workspace's view,
// each filter key it has, an empty one kept.
func viewOf(v domain.View) gen.View {
	return gen.View{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: orNull(v.ProjectID), Name: v.Name, Description: v.Description,
		LogoProps: logoOut(v.LogoProps), Filters: filtersOut(v.Filters), DisplayFilters: displayFiltersOut(v.DisplayFilters),
		DisplayProperties: displayPropertiesOut(v.DisplayProperties), OwnedByID: v.OwnedByID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt}
}

// filtersIn is f's thirteen keys as the domain holds them.
func filtersIn(f gen.ViewFilters) domain.Filters {
	return domain.Filters{ListFilters: shared.ListFilters{StateID: atoms(f.StateID), StateGroup: atoms(f.StateGroup), Priority: atoms(f.Priority),
		LabelID: atoms(f.LabelID), AssigneeID: atoms(f.AssigneeID), CreatedByID: atoms(f.CreatedByID), MentionID: atoms(f.MentionID),
		SubscriberID: atoms(f.SubscriberID), StartDate: atoms(f.StartDate), TargetDate: atoms(f.TargetDate)},
		CycleID: atoms(f.CycleID), ModuleID: atoms(f.ModuleID), ProjectID: atoms(f.ProjectID)}
}

// filtersOut is f as the API shows it.
func filtersOut(f domain.Filters) gen.ViewFilters {
	return gen.ViewFilters{StateID: array(f.StateID), StateGroup: array(f.StateGroup), Priority: array(f.Priority), LabelID: array(f.LabelID),
		AssigneeID: array(f.AssigneeID), CreatedByID: array(f.CreatedByID), MentionID: array(f.MentionID), SubscriberID: array(f.SubscriberID),
		StartDate: array(f.StartDate), TargetDate: array(f.TargetDate), CycleID: array(f.CycleID), ModuleID: array(f.ModuleID),
		ProjectID: array(f.ProjectID)}
}

// atoms is a filter key's atoms: nil when it is left out, an empty slice
// when it is given empty.
func atoms(v *[]string) []string {
	if v == nil {
		return nil
	}
	return *v
}

// array is a filter key's atoms as the API shows them: left out when nil.
func array(v []string) *[]string {
	if v == nil {
		return nil
	}
	return &v
}

// logoIn is l as the domain holds it, in_use unchecked (the domain checks
// it).
func logoIn(l apigen.LogoProps) shared.LogoProps {
	var out shared.LogoProps
	if l.InUse != nil {
		inUse := string(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &shared.LogoEmoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &shared.LogoIcon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}

// logoOut is l as the API shows it: {} for none.
func logoOut(l shared.LogoProps) apigen.LogoProps {
	var out apigen.LogoProps
	if l.InUse != nil {
		inUse := apigen.LogoPropsInUse(*l.InUse)
		out.InUse = &inUse
	}
	if e := l.Emoji; e != nil {
		out.Emoji = &apigen.LogoEmoji{Value: e.Value, URL: e.URL}
	}
	if i := l.Icon; i != nil {
		out.Icon = &apigen.LogoIcon{Name: i.Name, Color: i.Color, BackgroundColor: i.BackgroundColor}
	}
	return out
}

// displayFiltersIn is d as the use case takes it, nil when it is left out.
func displayFiltersIn(d *apigen.IssueListDisplayFilters) *shared.ListDisplayFilters {
	if d == nil {
		return nil
	}
	return &shared.ListDisplayFilters{Layout: d.Layout, GroupBy: grouping(d.GroupBy), SubGroupBy: grouping(d.SubGroupBy), OrderBy: d.OrderBy,
		SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: shared.ListCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
}

// displayPropertiesIn is v as the use case takes it, nil when it is left
// out.
func displayPropertiesIn(v *apigen.IssueListDisplayProperties) *shared.ListDisplayProperties {
	if v == nil {
		return nil
	}
	return &shared.ListDisplayProperties{Assignee: v.Assignee, StartDate: v.StartDate, DueDate: v.DueDate, Labels: v.Labels, Key: v.Key,
		Priority: v.Priority, State: v.State, SubIssueCount: v.SubIssueCount, Link: v.Link, AttachmentCount: v.AttachmentCount, CreatedOn: v.CreatedOn,
		UpdatedOn: v.UpdatedOn, Cycle: v.Cycle, Modules: v.Modules}
}

// displayFiltersOut is d as the API shows it: null for no grouping.
func displayFiltersOut(d shared.ListDisplayFilters) apigen.IssueListDisplayFilters {
	return apigen.IssueListDisplayFilters{Layout: d.Layout, GroupBy: orNull(d.GroupBy), SubGroupBy: orNull(d.SubGroupBy), OrderBy: d.OrderBy,
		SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: apigen.IssueListCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
}

// displayPropertiesOut is v as the API shows it.
func displayPropertiesOut(v shared.ListDisplayProperties) apigen.IssueListDisplayProperties {
	return apigen.IssueListDisplayProperties{Assignee: v.Assignee, StartDate: v.StartDate, DueDate: v.DueDate, Labels: v.Labels, Key: v.Key,
		Priority: v.Priority, State: v.State, SubIssueCount: v.SubIssueCount, Link: v.Link, AttachmentCount: v.AttachmentCount, CreatedOn: v.CreatedOn,
		UpdatedOn: v.UpdatedOn, Cycle: v.Cycle, Modules: v.Modules}
}

// grouping is a grouping word, nil for null.
func grouping(v nullable.Nullable[string]) *string {
	if v.IsNull() {
		return nil
	}
	s := v.MustGet()
	return &s
}

// orNull is v as a nullable value: null for nil, never left out.
func orNull[T any](v *T) nullable.Nullable[T] {
	if v == nil {
		return nullable.NewNullNullable[T]()
	}
	return nullable.NewNullableWithValue(*v)
}
