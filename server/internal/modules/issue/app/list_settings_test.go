package app_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueListSettings(t *testing.T) {
	validator := app.ListSettings{}
	display := func() app.ListDisplayFilters {
		return app.ListDisplayFilters{Layout: "list", OrderBy: "-created_at", Calendar: app.ListCalendar{Layout: "month"}}
	}
	check := func(t *testing.T, filters map[string][]string, d *app.ListDisplayFilters, properties map[string]bool, want ...string) {
		t.Helper()
		got := validator.Validate(filters, d, properties)
		paths := make([]string, 0, len(got))
		for _, e := range got {
			paths = append(paths, e.Field)
			if e.Code != shared.FieldNotAllowed {
				t.Errorf("error code = %+v", e)
			}
		}
		if len(paths) != len(want) || (len(want) > 0 && !reflect.DeepEqual(paths, want)) {
			t.Errorf("errors = %v, want %v", got, want)
		}
	}
	check(t, nil, nil, nil)
	check(t, map[string][]string{}, nil, nil)
	id := "0199a2b4-0000-7000-8000-000000000001"
	for _, field := range strings.Fields("state_id state_group priority label_id assignee_id created_by_id mention_id subscriber_id start_date target_date") {
		check(t, map[string][]string{field: {}}, nil, nil)
		for _, value := range []string{id, "me", "none", "high", "started", "2026-10-25", "..2026-10-25", "2026-10-25..", "2026-10-24..2026-10-25", "2026-02-30", "2026-10-25..2026-10-24", "..", "future", id + "," + id} {
			t.Run(field+"/"+value, func(t *testing.T) {
				var want []string
				if !domain.ListFilterValueAllowed(field, value) {
					want = []string{"filters." + field + "[0]"}
				}
				check(t, map[string][]string{field: {value}}, nil, nil, want...)
			})
		}
	}
	// Independent refusals supplement the shared-vocabulary parity cases.
	check(t, map[string][]string{"label_id": {"bad-id", id + "," + id}, "start_date": {"none", "2026-10-25"}}, nil, nil, "filters.label_id[0]", "filters.label_id[1]", "filters.start_date[0]")
	check(t, map[string][]string{"target_date": {"none", "none"}}, nil, nil, "filters.target_date[0]", "filters.target_date[1]")
	check(t, map[string][]string{"start_date": {"2026-10-01..2026-10-02", "2026-10-04..2026-10-05"}}, nil, nil)
	check(t, map[string][]string{"sub_issue": {}, "project_id": {}, "archived": {}}, nil, nil, "filters.archived", "filters.project_id", "filters.sub_issue")
	for _, word := range strings.Fields("created_at updated_at start_date target_date priority state_name label_name assignee_name sub_issues_count sort_order") {
		for _, prefix := range []string{"", "-"} {
			d := display()
			d.OrderBy = prefix + word
			if d.OrderBy == "-sort_order" {
				check(t, nil, &d, nil, "display_filters.order_by")
			} else {
				check(t, nil, &d, nil)
			}
		}
	}
	for _, layout := range strings.Fields("list kanban calendar spreadsheet") {
		for _, calendar := range []string{"month", "week"} {
			d := display()
			d.Layout = layout
			d.Calendar.Layout = calendar
			check(t, nil, &d, nil)
		}
	}
	groups := strings.Fields("state_id state_group priority label_id assignee_id created_by_id target_date")
	for _, group := range groups {
		d := display()
		d.GroupBy = &group
		check(t, nil, &d, nil)
		for _, subgroup := range groups {
			d.SubGroupBy = &subgroup
			if subgroup == group {
				check(t, nil, &d, nil, "display_filters.sub_group_by")
			} else {
				check(t, nil, &d, nil)
			}
		}
	}
	for _, future := range []string{"cycle_id", "module_id", "project_id", "future"} {
		d := display()
		d.GroupBy = &future
		check(t, nil, &d, nil, "display_filters.group_by")
		d = display()
		d.SubGroupBy = &future
		check(t, nil, &d, nil, "display_filters.sub_group_by")
	}
	d := display()
	group := "state_id"
	d.SubGroupBy = &group
	check(t, nil, &d, nil, "display_filters.sub_group_by")
	d = display()
	d.Layout = "future"
	d.OrderBy = "link_count"
	d.Calendar.Layout = "day"
	check(t, nil, &d, nil, "display_filters.layout", "display_filters.order_by", "display_filters.calendar.layout")
	properties := map[string]bool{}
	for i, key := range strings.Fields("assignee start_date due_date labels key priority state sub_issue_count link created_on updated_on") {
		properties[key] = i%2 == 0
	}
	check(t, nil, nil, properties)
	check(t, nil, nil, map[string]bool{"link_count": true}, "display_properties.link_count")
	filters := map[string][]string{"assignee_id": {"me", id}, "label_id": {}}
	before := fmt.Sprintf("%#v", filters)
	check(t, filters, nil, nil)
	if after := fmt.Sprintf("%#v", filters); after != before {
		t.Fatalf("validator changed input: %s", after)
	}
}
