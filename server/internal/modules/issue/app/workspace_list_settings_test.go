package app_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// settingsValidator is ListSettings or WorkspaceListSettings.
type settingsValidator interface {
	Validate(map[string][]string, *shared.ListDisplayFilters) []shared.FieldError
}

// refusedPaths checks that every problem is not_allowed and returns their
// fields, in the order reported.
func refusedPaths(t *testing.T, v settingsValidator, filters map[string][]string, d *shared.ListDisplayFilters) []string {
	t.Helper()
	var paths []string
	for _, e := range v.Validate(filters, d) {
		if e.Code != shared.FieldNotAllowed {
			t.Errorf("error code = %+v", e)
		}
		paths = append(paths, e.Field)
	}
	return paths
}

// The saved filter keys of the two kinds of views (M7/P3b spec 2.3, re-90
// I1): a project's 12, the 10 every page shares and cycle_id, module_id; a
// workspace's 11, the 10 and project_id. Neither saves sub_issue, a display
// word, nor archived; the workspace's has no cycle, module or relation.
func TestSavedFilterKeysByPage(t *testing.T) {
	var common []string
	rt := reflect.TypeFor[shared.ListFilters]()
	for i := range rt.NumField() {
		common = append(common, strings.Split(rt.Field(i).Tag.Get("json"), ",")[0])
	}
	if len(common) != 10 {
		t.Fatalf("shared.ListFilters has %d keys %v; want 10", len(common), common)
	}
	candidates := append(slices.Clone(common), "cycle_id", "module_id", "project_id", "sub_issue", "archived", "relation", "future")
	for _, tc := range []struct {
		name      string
		validator settingsValidator
		want      []string
	}{
		{"project", app.ListSettings{}, append(slices.Clone(common), "cycle_id", "module_id")},
		{"workspace", app.WorkspaceListSettings{}, append(slices.Clone(common), "project_id")},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var accepted []string
			for _, key := range candidates {
				switch got := refusedPaths(t, tc.validator, map[string][]string{key: {}}, nil); {
				case len(got) == 0:
					accepted = append(accepted, key)
				case !reflect.DeepEqual(got, []string{"filters." + key}):
					t.Errorf("filters {%s: []}: refused %v", key, got)
				}
			}
			if !reflect.DeepEqual(accepted, tc.want) {
				t.Errorf("accepted keys = %v (%d); want %v (%d)", accepted, len(accepted), tc.want, len(tc.want))
			}
		})
	}
}

// A workspace view's saved words (M7/P3b spec 2.4, DP-9 (a)): the
// workspace list's filters, valued as every page's; a flat spreadsheet or
// list, ordered and with a calendar as every page's.
func TestWorkspaceListSettings(t *testing.T) {
	v := app.WorkspaceListSettings{}
	display := func() shared.ListDisplayFilters {
		return shared.ListDisplayFilters{Layout: "spreadsheet", OrderBy: "-created_at", Calendar: shared.ListCalendar{Layout: "month"}}
	}
	check := func(t *testing.T, filters map[string][]string, d *shared.ListDisplayFilters, want ...string) {
		t.Helper()
		if got := refusedPaths(t, v, filters, d); !reflect.DeepEqual(got, want) && (len(got) > 0 || len(want) > 0) {
			t.Errorf("refused %v; want %v", got, want)
		}
	}
	check(t, nil, nil)
	check(t, map[string][]string{}, nil)
	id := "0199a2b4-0000-7000-8000-000000000001"
	for _, key := range strings.Fields("project_id state_id state_group priority label_id assignee_id created_by_id mention_id subscriber_id start_date target_date") {
		for _, value := range []string{id, "me", "none", "high", "started", "true", "2026-10-25", "..2026-10-25", "2026-10-24..2026-10-25", "2026-02-30", "..", "future", id + "," + id} {
			t.Run(key+"/"+value, func(t *testing.T) {
				var want []string
				if !domain.ListFilterValueAllowed(key, value) {
					want = []string{"filters." + key + "[0]"}
				}
				check(t, map[string][]string{key: {value}}, nil, want...)
			})
		}
	}
	check(t, map[string][]string{"project_id": {"none", id, "me"}, "target_date": {"none", "2026-10-25"}}, nil,
		"filters.project_id[0]", "filters.project_id[2]", "filters.target_date[0]")
	check(t, map[string][]string{"sub_issue": {"true"}, "cycle_id": {id}, "module_id": {"none"}, "archived": {"false"}, "relation": {}, "project_id": {id}}, nil,
		"filters.archived", "filters.cycle_id", "filters.module_id", "filters.relation", "filters.sub_issue")

	for _, layout := range []string{"spreadsheet", "list"} {
		for _, calendar := range []string{"month", "week"} {
			for _, subIssue := range []bool{true, false} {
				d := display()
				d.Layout, d.Calendar.Layout, d.SubIssue, d.ShowEmptyGroups = layout, calendar, subIssue, !subIssue
				check(t, nil, &d)
			}
		}
	}
	for _, layout := range []string{"kanban", "calendar", "gantt_chart", "future", ""} {
		d := display()
		d.Layout = layout
		check(t, nil, &d, "display_filters.layout")
	}
	for _, word := range strings.Fields("created_at updated_at start_date target_date priority state_name label_name assignee_name cycle_name module_name sub_issues_count link_count attachment_count sort_order") {
		for _, prefix := range []string{"", "-"} {
			d := display()
			d.OrderBy = prefix + word
			if domain.ListOrderAllowed(domain.ListOrder(d.OrderBy)) {
				check(t, nil, &d)
			} else {
				check(t, nil, &d, "display_filters.order_by")
			}
		}
	}
	// Flat only: every group a page has, and none, is refused.
	for _, group := range strings.Fields("state_id state_group priority label_id assignee_id created_by_id cycle_id module_id target_date project_id future") {
		d := display()
		d.GroupBy = &group
		check(t, nil, &d, "display_filters.group_by")
		d = display()
		d.SubGroupBy = &group
		check(t, nil, &d, "display_filters.sub_group_by")
		other := "priority"
		d.GroupBy = &other
		check(t, nil, &d, "display_filters.group_by", "display_filters.sub_group_by")
	}
	d := display()
	group, subgroup := "state_id", "priority"
	d.Layout, d.GroupBy, d.SubGroupBy, d.OrderBy, d.Calendar.Layout = "kanban", &group, &subgroup, "-sort_order", "day"
	check(t, map[string][]string{"sub_issue": {}, "priority": {"urgent", "high"}}, &d, "filters.sub_issue",
		"display_filters.layout", "display_filters.group_by", "display_filters.sub_group_by", "display_filters.order_by", "display_filters.calendar.layout")

	filters := map[string][]string{"assignee_id": {"me", id}, "label_id": {}, "project_id": {id}}
	before := fmt.Sprintf("%#v", filters)
	check(t, filters, nil)
	if after := fmt.Sprintf("%#v", filters); after != before {
		t.Fatalf("validator changed input: %s", after)
	}
}
