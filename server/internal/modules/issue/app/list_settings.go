package app

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListSettings validates a project list's saved words — a project's, a
// cycle's and a module's page, and a project's views — without a query,
// actor or directory. Its zero value is ready to use and never changes its
// inputs.
type ListSettings struct{}

// Validate checks the words by projectSettings (savedSettings.validate).
func (ListSettings) Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
	return projectSettings.validate(filters, display)
}

// WorkspaceListSettings validates a workspace view's saved words (M7/P3b
// spec 2.4): the workspace list's filters (M7/P3a), shown flat as a
// spreadsheet or a list. Like ListSettings, its zero value is ready to use
// and never changes its inputs.
type WorkspaceListSettings struct{}

// Validate checks the words by workspaceSettings (savedSettings.validate).
func (WorkspaceListSettings) Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
	return workspaceSettings.validate(filters, display)
}

// savedSettings are one page's saved words: the filters, groups and
// sub-groups of its list scope, and its layouts.
type savedSettings struct {
	scope   domain.ListScope
	layouts []string
}

var (
	projectSettings = savedSettings{scope: domain.ProjectList, layouts: []string{"list", "kanban", "calendar", "spreadsheet"}}
	// The workspace's list has no group (M7/P3a spec 2.2), so a group or a
	// sub-group is refused; Plane's workspace views are spreadsheets.
	workspaceSettings = savedSettings{scope: domain.WorkspaceList, layouts: []string{"spreadsheet", "list"}}
)

// validate accepts nil top-level fields as omitted and empty filter arrays
// as no filtering. Saved filters hold neither archived nor sub_issue: the
// list's sub_issue is saved as display_filters.sub_issue. Errors are
// deterministic: sorted filter keys and original element order, then
// display fields. Display properties are not checked here: they are
// booleans whose keys shared.ListDisplayProperties fixes.
func (s savedSettings) validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
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
		if !domain.ListFilterAllowed(key, s.scope) || key == "archived" || key == "sub_issue" {
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
		if !slices.Contains(s.layouts, d.Layout) {
			refuse("display_filters.layout")
		}
		if d.GroupBy != nil && !domain.ListGroupAllowed(domain.ListGroup(*d.GroupBy), s.scope) {
			refuse("display_filters.group_by")
		}
		if d.SubGroupBy != nil && (!domain.ListSubGroupAllowed(domain.ListGroup(*d.SubGroupBy), s.scope) || d.GroupBy == nil || *d.SubGroupBy == *d.GroupBy) {
			refuse("display_filters.sub_group_by")
		}
		if !domain.ListOrderAllowed(domain.ListOrder(d.OrderBy)) {
			refuse("display_filters.order_by")
		}
		if d.Calendar.Layout != "month" && d.Calendar.Layout != "week" {
			refuse("display_filters.calendar.layout")
		}
	}
	return found
}
