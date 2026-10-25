package app

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListCalendar and ListDisplayFilters are issue app boundary values. The
// composition root converts the consuming module's own values into these.
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

// ListSettings validates saved words without a query, actor or directory.
// Its zero value is ready to use and never changes its inputs.
type ListSettings struct{}

// Validate accepts nil top-level fields as omitted and empty filter arrays
// as no filtering. Errors are deterministic: sorted filter keys and original
// element order, display fields, then sorted property keys.
func (ListSettings) Validate(filters map[string][]string, display *ListDisplayFilters, properties map[string]bool) []shared.FieldError {
	var found []shared.FieldError
	refuse := func(field string) {
		found = append(found, shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is not allowed"})
	}
	keys := make([]string, 0, len(filters))
	for key := range filters {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		if !domain.ListFilterAllowed(key, domain.ProjectList) || key == "archived" || key == "sub_issue" {
			refuse("filters." + key)
			continue
		}
		values := filters[key]
		for i, value := range values {
			allowed := domain.ListFilterValueAllowed(key, value)
			if (key == "start_date" || key == "target_date") && value == "none" && len(values) > 1 {
				allowed = false
			}
			if !allowed {
				refuse(fmt.Sprintf("filters.%s[%d]", key, i))
			}
		}
	}
	if d := display; d != nil {
		if !slices.Contains([]string{"list", "kanban", "calendar", "spreadsheet"}, d.Layout) {
			refuse("display_filters.layout")
		}
		if d.GroupBy != nil && !domain.ListGroupAllowed(domain.ListGroup(*d.GroupBy), domain.ProjectList) {
			refuse("display_filters.group_by")
		}
		if d.SubGroupBy != nil && (!domain.ListSubGroupAllowed(domain.ListGroup(*d.SubGroupBy), domain.ProjectList) || d.GroupBy == nil || *d.SubGroupBy == *d.GroupBy) {
			refuse("display_filters.sub_group_by")
		}
		if !domain.ListOrderAllowed(domain.ListOrder(d.OrderBy)) {
			refuse("display_filters.order_by")
		}
		if d.Calendar.Layout != "month" && d.Calendar.Layout != "week" {
			refuse("display_filters.calendar.layout")
		}
	}
	keys = keys[:0]
	for key := range properties {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		switch key {
		case "assignee", "start_date", "due_date", "labels", "key", "priority", "state", "sub_issue_count", "link", "created_on", "updated_on":
		default:
			refuse("display_properties." + key)
		}
	}
	return found
}
