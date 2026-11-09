package domain_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Defaults and whole-field replacement are independent of storage and HTTP.
func TestProjectIssuePreferencesValues(t *testing.T) {
	want := domain.Preferences{
		Navigation: domain.Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}, SortOrder: 65535,
		Filters:           domain.Filters{},
		DisplayFilters:    shared.ListDisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: shared.ListCalendar{Layout: "month"}},
		DisplayProperties: shared.ListDisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true, SubIssueCount: true, Link: true, AttachmentCount: true, CreatedOn: true, UpdatedOn: true, Cycle: true, Modules: true},
	}
	p := domain.DefaultPreferences()
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("defaults = %+v, want %+v", p, want)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{ListFilters: shared.ListFilters{LabelID: []string{"none"}, Priority: []string{"high"}}, CycleID: []string{"none"}}})
	want.Filters = domain.Filters{ListFilters: shared.ListFilters{LabelID: []string{"none"}, Priority: []string{"high"}}, CycleID: []string{"none"}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("first filters = %+v", p)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{ListFilters: shared.ListFilters{LabelID: []string{}}}})
	want.Filters = domain.Filters{ListFilters: shared.ListFilters{LabelID: []string{}}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("replacement/empty array = %+v", p)
	}
	display := shared.ListDisplayFilters{Layout: "kanban", OrderBy: "sort_order", Calendar: shared.ListCalendar{Layout: "week"}}
	p = p.Apply(domain.PreferencesPatch{DisplayFilters: &display, DisplayProperties: &shared.ListDisplayProperties{Key: true}})
	want.DisplayFilters = display
	want.DisplayProperties = shared.ListDisplayProperties{Key: true}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("false flags and omitted filters = %+v", p)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{}})
	want.Filters = domain.Filters{}
	if !reflect.DeepEqual(p, want) || !reflect.DeepEqual(p.Apply(domain.PreferencesPatch{}), want) {
		t.Fatalf("clear/omit = %+v", p)
	}
}

// The project's filters are the shared keys and its own two, each by its
// JSON name in the map the settings port checks: a nil slice is left out,
// an empty one kept.
func TestProjectFiltersByJSONName(t *testing.T) {
	f := domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"low"}}, CycleID: []string{}, ModuleID: []string{"none"}}
	if m, want := f.Map(), map[string][]string{"priority": {"low"}, "cycle_id": {}, "module_id": {"none"}}; !reflect.DeepEqual(m, want) || m["cycle_id"] == nil {
		t.Fatalf("Map() = %#v, want %#v", m, want)
	}
	if m := (domain.Filters{}).Map(); m == nil || len(m) != 0 {
		t.Fatalf("no keys: %#v, want an empty map", m)
	}
	all := domain.Filters{ListFilters: shared.ListFilters{StateID: []string{"1"}, StateGroup: []string{"2"}, Priority: []string{"3"}, LabelID: []string{"4"}, AssigneeID: []string{"5"},
		CreatedByID: []string{"6"}, MentionID: []string{"7"}, SubscriberID: []string{"8"}, StartDate: []string{"9"}, TargetDate: []string{"10"}}, CycleID: []string{"11"}, ModuleID: []string{"12"}}
	b, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	var byJSON map[string][]string
	if err := json.Unmarshal(b, &byJSON); err != nil || len(byJSON) != 12 || !reflect.DeepEqual(all.Map(), byJSON) {
		t.Fatalf("Map() = %v; JSON %s", all.Map(), b)
	}
}
