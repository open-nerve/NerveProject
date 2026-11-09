package app

import (
	"fmt"
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListSettings validates saved words without a query, actor or directory.
// Its zero value is ready to use and never changes its inputs.
type ListSettings struct{}

// Validate accepts nil top-level fields as omitted and empty filter arrays
// as no filtering. Errors are deterministic: sorted filter keys and original
// element order, then display fields. Display properties are not checked
// here: they are booleans whose keys shared.ListDisplayProperties fixes.
func (ListSettings) Validate(filters map[string][]string, display *shared.ListDisplayFilters) []shared.FieldError {
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
	return found
}
