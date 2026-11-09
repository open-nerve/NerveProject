package domain_test

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The two actions of a page's own display settings (M6 P3b spec 2.3) are the module's; the module page's is a
// module action, so TestModuleAction holds it, though the settings' path never asks the toggles (R6).
func TestPreferencesActions(t *testing.T) {
	if domain.ActionCyclePreferences != "cycle.preferences" || domain.ActionModulePreferences != "module.preferences" {
		t.Fatalf("actions %s, %s", domain.ActionCyclePreferences, domain.ActionModulePreferences)
	}
	for _, a := range []shared.Action{domain.ActionCyclePreferences, domain.ActionModulePreferences} {
		if !slices.Contains(domain.Actions(), a) {
			t.Errorf("Actions() = %v, want %s among them", domain.Actions(), a)
		}
	}
	if domain.ModuleAction(domain.ActionCyclePreferences) || !domain.ModuleAction(domain.ActionModulePreferences) {
		t.Error("want module.preferences a module action, cycle.preferences not")
	}
}

// A page's filters are the shared ten keys and the other page's (R5, R8): a cycle's page has module_id and no cycle_id,
// a module's page cycle_id and no module_id. Map gives each key by its JSON name, a nil slice left out and an empty one
// kept, so its keys are exactly the JSON's.
func TestPageFiltersByJSONName(t *testing.T) {
	all := shared.ListFilters{StateID: []string{"1"}, StateGroup: []string{"2"}, Priority: []string{"3"}, LabelID: []string{"4"}, AssigneeID: []string{"5"},
		CreatedByID: []string{"6"}, MentionID: []string{"7"}, SubscriberID: []string{"8"}, StartDate: []string{"9"}, TargetDate: []string{"10"}}
	some := shared.ListFilters{Priority: []string{"low"}}
	t.Run("cycle", func(t *testing.T) {
		pageFiltersByJSONName(t, domain.CycleFilters{ListFilters: all, ModuleID: []string{"11"}}, domain.CycleFilters{ListFilters: some, ModuleID: []string{}},
			"module_id", "cycle_id")
	})
	t.Run("module", func(t *testing.T) {
		pageFiltersByJSONName(t, domain.ModuleFilters{ListFilters: all, CycleID: []string{"11"}}, domain.ModuleFilters{ListFilters: some, CycleID: []string{}},
			"cycle_id", "module_id")
	})
}

// pageFiltersByJSONName checks a page's filters: all, every key given, by their JSON names; some, priority low and an
// empty own; none, the zero filters. own is the other page's key, page the page's own.
func pageFiltersByJSONName[F domain.PageFilters](t *testing.T, all, some F, own, page string) {
	t.Helper()
	b, err := json.Marshal(all)
	if err != nil {
		t.Fatal(err)
	}
	var byJSON map[string][]string
	if err := json.Unmarshal(b, &byJSON); err != nil || len(byJSON) != 11 || !reflect.DeepEqual(all.Map(), byJSON) {
		t.Fatalf("Map() = %v; JSON %s; want the same 11 keys", all.Map(), b)
	}
	if _, ok := byJSON[own]; !ok {
		t.Errorf("JSON %s lacks %s", b, own)
	}
	if _, ok := byJSON[page]; ok {
		t.Errorf("JSON %s has the page's own key %s", b, page)
	}
	if m, want := some.Map(), map[string][]string{"priority": {"low"}, own: {}}; !reflect.DeepEqual(m, want) || m[own] == nil {
		t.Errorf("Map() = %#v, want %#v", m, want)
	}
	var none F
	if m := none.Map(); m == nil || len(m) != 0 {
		t.Errorf("no keys: %#v, want an empty map", m)
	}
}

// The defaults are the columns' (00046, 00047): no filter, stored as {}, and the display defaults every page has. A
// change replaces each field it gives whole, an empty array kept and an empty filter clearing them, and leaves the
// others.
func TestPagePreferencesValues(t *testing.T) {
	want := domain.CyclePreferences{
		DisplayFilters:    shared.ListDisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: shared.ListCalendar{Layout: "month"}},
		DisplayProperties: shared.ListDisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true, SubIssueCount: true, Link: true, AttachmentCount: true, CreatedOn: true, UpdatedOn: true, Cycle: true, Modules: true},
	}
	p := domain.DefaultPreferences[domain.CycleFilters]()
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("defaults = %+v, want %+v", p, want)
	}
	if b, err := json.Marshal(p.Filters); err != nil || string(b) != "{}" {
		t.Fatalf("default filters as JSON = %s, %v; want {}", b, err)
	}
	if m := domain.DefaultPreferences[domain.ModuleFilters](); m.DisplayFilters != want.DisplayFilters || m.DisplayProperties != want.DisplayProperties ||
		!reflect.DeepEqual(m.Filters, domain.ModuleFilters{}) {
		t.Fatalf("a module page's defaults = %+v", m)
	}
	filters := domain.CycleFilters{ListFilters: shared.ListFilters{LabelID: []string{"none"}, Priority: []string{"high"}}, ModuleID: []string{"none"}}
	p = p.Apply(domain.CyclePreferencesPatch{Filters: &filters})
	want.Filters = filters
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("first filters = %+v", p)
	}
	p = p.Apply(domain.CyclePreferencesPatch{Filters: &domain.CycleFilters{ListFilters: shared.ListFilters{LabelID: []string{}}}})
	want.Filters = domain.CycleFilters{ListFilters: shared.ListFilters{LabelID: []string{}}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("replacement, an empty array = %+v", p)
	}
	display := shared.ListDisplayFilters{Layout: "kanban", OrderBy: "sort_order", Calendar: shared.ListCalendar{Layout: "week"}}
	p = p.Apply(domain.CyclePreferencesPatch{DisplayFilters: &display, DisplayProperties: &shared.ListDisplayProperties{Key: true}})
	want.DisplayFilters, want.DisplayProperties = display, shared.ListDisplayProperties{Key: true}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("display filters and properties, the filters left = %+v", p)
	}
	p = p.Apply(domain.CyclePreferencesPatch{Filters: &domain.CycleFilters{}})
	want.Filters = domain.CycleFilters{}
	if !reflect.DeepEqual(p, want) || !reflect.DeepEqual(p.Apply(domain.CyclePreferencesPatch{}), want) {
		t.Fatalf("cleared filters, then an empty change = %+v", p)
	}
}
