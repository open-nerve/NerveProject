package app

import (
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListCalendar and ListDisplayFilters are this module's saved-list boundary values.
type ListCalendar struct {
	Layout       string
	ShowWeekends bool
}
type ListDisplayFilters struct {
	Layout                    string
	GroupBy, SubGroupBy       *string
	OrderBy                   string
	SubIssue, ShowEmptyGroups bool
	Calendar                  ListCalendar
}

// IssueListSettings checks saved words without querying a directory or interpreting an actor.
// Nil top-level values mean omitted; a nonnil empty filters map means clear.
type IssueListSettings interface {
	Validate(filters map[string][]string, display *ListDisplayFilters, properties map[string]bool) []shared.FieldError
}

// checkListSettings preserves navigation's error order, then the settings port's order.
func checkListSettings(p domain.PreferencesPatch, settings IssueListSettings) error {
	var fields []shared.FieldError
	if err := domain.CheckPreferencesPatch(p); err != nil {
		var invalid *shared.Error
		if !errors.As(err, &invalid) || invalid.Code != shared.CodeValidationFailed {
			return err
		}
		fields = append(fields, invalid.Fields...)
	}
	var filters map[string][]string
	if f := p.Filters; f != nil {
		filters = make(map[string][]string)
		for _, entry := range []struct {
			key    string
			values []string
		}{
			{"state_id", f.StateID}, {"state_group", f.StateGroup}, {"priority", f.Priority}, {"label_id", f.LabelID}, {"assignee_id", f.AssigneeID},
			{"created_by_id", f.CreatedByID}, {"mention_id", f.MentionID}, {"subscriber_id", f.SubscriberID}, {"start_date", f.StartDate}, {"target_date", f.TargetDate},
		} {
			if entry.values != nil {
				filters[entry.key] = entry.values
			}
		}
	}
	var display *ListDisplayFilters
	if d := p.DisplayFilters; d != nil {
		display = &ListDisplayFilters{Layout: d.Layout, GroupBy: d.GroupBy, SubGroupBy: d.SubGroupBy, OrderBy: d.OrderBy,
			SubIssue: d.SubIssue, ShowEmptyGroups: d.ShowEmptyGroups, Calendar: ListCalendar{Layout: d.Calendar.Layout, ShowWeekends: d.Calendar.ShowWeekends}}
	}
	var properties map[string]bool
	if d := p.DisplayProperties; d != nil {
		properties = map[string]bool{"assignee": d.Assignee, "start_date": d.StartDate, "due_date": d.DueDate, "labels": d.Labels, "key": d.Key,
			"priority": d.Priority, "state": d.State, "sub_issue_count": d.SubIssueCount, "link": d.Link, "created_on": d.CreatedOn, "updated_on": d.UpdatedOn}
	}
	fields = append(fields, settings.Validate(filters, display, properties)...)
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	return nil
}
