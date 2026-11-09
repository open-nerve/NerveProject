package httpadapter

import (
	"context"

	"github.com/oapi-codegen/nullable"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apigen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetCyclePreferences serves GET /api/v0/me/cycles/{cycle_id}/preferences.
func (h handler) GetCyclePreferences(ctx context.Context, req gen.GetCyclePreferencesRequestObject) (gen.GetCyclePreferencesResponseObject, error) {
	p, err := h.uc.GetCyclePreferences.Execute(ctx, req.CycleID)
	if err != nil {
		return nil, err
	}
	return gen.GetCyclePreferences200JSONResponse(cyclePreferences(p)), nil
}

// UpdateCyclePreferences serves PATCH
// /api/v0/me/cycles/{cycle_id}/preferences: the fields the body names go to
// the use case, each whole, with their words as given (the use case checks
// them); a filter key left out stays apart from an empty one.
func (h handler) UpdateCyclePreferences(ctx context.Context, req gen.UpdateCyclePreferencesRequestObject) (gen.UpdateCyclePreferencesResponseObject, error) {
	b := req.Body
	in := domain.CyclePreferencesPatch{DisplayFilters: displayFiltersIn(b.DisplayFilters), DisplayProperties: displayPropertiesIn(b.DisplayProperties)}
	if f := b.Filters; f != nil {
		in.Filters = &domain.CycleFilters{ListFilters: shared.ListFilters{StateID: atoms(f.StateID), StateGroup: atoms(f.StateGroup), Priority: atoms(f.Priority),
			LabelID: atoms(f.LabelID), AssigneeID: atoms(f.AssigneeID), CreatedByID: atoms(f.CreatedByID), MentionID: atoms(f.MentionID),
			SubscriberID: atoms(f.SubscriberID), StartDate: atoms(f.StartDate), TargetDate: atoms(f.TargetDate)}, ModuleID: atoms(f.ModuleID)}
	}
	p, err := h.uc.UpdateCyclePreferences.Execute(ctx, req.CycleID, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateCyclePreferences200JSONResponse(cyclePreferences(p)), nil
}

// GetModulePreferences serves GET /api/v0/me/modules/{module_id}/preferences.
func (h handler) GetModulePreferences(ctx context.Context, req gen.GetModulePreferencesRequestObject) (gen.GetModulePreferencesResponseObject, error) {
	p, err := h.uc.GetModulePreferences.Execute(ctx, req.ModuleID)
	if err != nil {
		return nil, err
	}
	return gen.GetModulePreferences200JSONResponse(modulePreferences(p)), nil
}

// UpdateModulePreferences serves PATCH
// /api/v0/me/modules/{module_id}/preferences, as UpdateCyclePreferences.
func (h handler) UpdateModulePreferences(ctx context.Context, req gen.UpdateModulePreferencesRequestObject) (gen.UpdateModulePreferencesResponseObject, error) {
	b := req.Body
	in := domain.ModulePreferencesPatch{DisplayFilters: displayFiltersIn(b.DisplayFilters), DisplayProperties: displayPropertiesIn(b.DisplayProperties)}
	if f := b.Filters; f != nil {
		in.Filters = &domain.ModuleFilters{ListFilters: shared.ListFilters{StateID: atoms(f.StateID), StateGroup: atoms(f.StateGroup), Priority: atoms(f.Priority),
			LabelID: atoms(f.LabelID), AssigneeID: atoms(f.AssigneeID), CreatedByID: atoms(f.CreatedByID), MentionID: atoms(f.MentionID),
			SubscriberID: atoms(f.SubscriberID), StartDate: atoms(f.StartDate), TargetDate: atoms(f.TargetDate)}, CycleID: atoms(f.CycleID)}
	}
	p, err := h.uc.UpdateModulePreferences.Execute(ctx, req.ModuleID, in)
	if err != nil {
		return nil, err
	}
	return gen.UpdateModulePreferences200JSONResponse(modulePreferences(p)), nil
}

// cyclePreferences is p as the API shows it: a filter key it has not left
// out, an empty one kept.
func cyclePreferences(p domain.CyclePreferences) gen.CyclePreferences {
	f := p.Filters
	return gen.CyclePreferences{Filters: gen.CycleIssueFilters{StateID: array(f.StateID), StateGroup: array(f.StateGroup), Priority: array(f.Priority),
		LabelID: array(f.LabelID), AssigneeID: array(f.AssigneeID), CreatedByID: array(f.CreatedByID), MentionID: array(f.MentionID),
		SubscriberID: array(f.SubscriberID), StartDate: array(f.StartDate), TargetDate: array(f.TargetDate), ModuleID: array(f.ModuleID)},
		DisplayFilters: displayFilters(p.DisplayFilters), DisplayProperties: displayProperties(p.DisplayProperties)}
}

// modulePreferences is p as the API shows it, as cyclePreferences.
func modulePreferences(p domain.ModulePreferences) gen.ModulePreferences {
	f := p.Filters
	return gen.ModulePreferences{Filters: gen.ModuleIssueFilters{StateID: array(f.StateID), StateGroup: array(f.StateGroup), Priority: array(f.Priority),
		LabelID: array(f.LabelID), AssigneeID: array(f.AssigneeID), CreatedByID: array(f.CreatedByID), MentionID: array(f.MentionID),
		SubscriberID: array(f.SubscriberID), StartDate: array(f.StartDate), TargetDate: array(f.TargetDate), CycleID: array(f.CycleID)},
		DisplayFilters: displayFilters(p.DisplayFilters), DisplayProperties: displayProperties(p.DisplayProperties)}
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

// grouping is a grouping word, nil for null.
func grouping(v nullable.Nullable[string]) *string {
	if v.IsNull() {
		return nil
	}
	s := v.MustGet()
	return &s
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

// displayFilters is d as the API shows it: null for no grouping.
func displayFilters(d shared.ListDisplayFilters) apigen.IssueListDisplayFilters {
	return apigen.IssueListDisplayFilters{Layout: d.Layout, GroupBy: orNull(d.GroupBy), SubGroupBy: orNull(d.SubGroupBy), OrderBy: d.OrderBy,
		SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: apigen.IssueListCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
}

// displayProperties is v as the API shows it.
func displayProperties(v shared.ListDisplayProperties) apigen.IssueListDisplayProperties {
	return apigen.IssueListDisplayProperties{Assignee: v.Assignee, StartDate: v.StartDate, DueDate: v.DueDate, Labels: v.Labels, Key: v.Key,
		Priority: v.Priority, State: v.State, SubIssueCount: v.SubIssueCount, Link: v.Link, AttachmentCount: v.AttachmentCount, CreatedOn: v.CreatedOn,
		UpdatedOn: v.UpdatedOn, Cycle: v.Cycle, Modules: v.Modules}
}
