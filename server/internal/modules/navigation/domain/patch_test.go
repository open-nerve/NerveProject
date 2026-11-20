package domain_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// stored is a view's content as a read gives it: every field set, the filters with an explicitly empty key.
func stored() domain.Content {
	return domain.Content{
		Name: "Mine", Description: "about",
		LogoProps: shared.LogoProps{InUse: ptr(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: ptr("star"), Color: ptr("#fff")}},
		Filters:   domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"low", "high"}, LabelID: []string{}}, CycleID: []string{"c"}},
		DisplayFilters: shared.ListDisplayFilters{Layout: "kanban", GroupBy: ptr("state"), SubGroupBy: ptr("priority"), OrderBy: "sort_order",
			SubIssue: true, Calendar: shared.ListCalendar{Layout: "week"}},
		DisplayProperties: shared.ListDisplayProperties{Key: true, State: true},
	}
}

// show is c as JSON, an explicitly empty filter apart from an omitted one.
func show(c domain.Content) string {
	data, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(data)
}

// A change replaces each top-level field it gives whole, never merging into it, and leaves the others (spec 2.4).
// What it changes is told on the content as stored, after its check: a field given as it is, in values equal to the
// stored ones, is no change, and {} changes nothing; the use case then answers the view as it is, without a write.
// An explicitly empty filter is not an omitted one, nor is an icon given empty the icon kept.
func TestPatchApply(t *testing.T) {
	s := stored()
	same := stored()
	tests := []struct {
		name    string
		patch   domain.Patch
		want    func(*domain.Content)
		changed bool
	}{
		{"{}", domain.Patch{}, func(*domain.Content) {}, false},
		{"every field as it is", domain.Patch{Name: &same.Name, Description: &same.Description, LogoProps: &same.LogoProps, Filters: &same.Filters,
			DisplayFilters: &same.DisplayFilters, DisplayProperties: &same.DisplayProperties}, func(*domain.Content) {}, false},
		{"the name as it is, once checked", domain.Patch{Name: ptr("  Mine ")}, func(*domain.Content) {}, false},
		{"name", domain.Patch{Name: ptr("Ours")}, func(c *domain.Content) { c.Name = "Ours" }, true},
		{"description", domain.Patch{Description: ptr("")}, func(c *domain.Content) { c.Description = "" }, true},
		{"no icon", domain.Patch{LogoProps: &shared.LogoProps{}}, func(c *domain.Content) { c.LogoProps = shared.LogoProps{} }, true},
		{"another icon color", domain.Patch{LogoProps: &shared.LogoProps{InUse: ptr(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: ptr("star"), Color: ptr("#000")}}},
			func(c *domain.Content) { c.LogoProps.Icon = &shared.LogoIcon{Name: ptr("star"), Color: ptr("#000")} }, true},
		{"filters replaced whole", domain.Patch{Filters: &domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"urgent"}}}},
			func(c *domain.Content) {
				c.Filters = domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"urgent"}}}
			}, true},
		{"no filter", domain.Patch{Filters: &domain.Filters{}}, func(c *domain.Content) { c.Filters = domain.Filters{} }, true},
		{"the empty filter omitted", domain.Patch{Filters: &domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"low", "high"}}, CycleID: []string{"c"}}},
			func(c *domain.Content) { c.Filters.LabelID = nil }, true},
		{"filter values in another order", domain.Patch{Filters: &domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"high", "low"}, LabelID: []string{}}, CycleID: []string{"c"}}},
			func(c *domain.Content) { c.Filters.Priority = []string{"high", "low"} }, true},
		{"display filters replaced whole", domain.Patch{DisplayFilters: ptr(shared.DefaultListDisplayFilters())},
			func(c *domain.Content) { c.DisplayFilters = shared.DefaultListDisplayFilters() }, true},
		{"ungrouped", domain.Patch{DisplayFilters: &shared.ListDisplayFilters{Layout: "kanban", OrderBy: "sort_order", SubIssue: true, Calendar: shared.ListCalendar{Layout: "week"}}},
			func(c *domain.Content) { c.DisplayFilters.GroupBy, c.DisplayFilters.SubGroupBy = nil, nil }, true},
		{"display properties replaced whole", domain.Patch{DisplayProperties: &shared.ListDisplayProperties{Key: true}},
			func(c *domain.Content) { c.DisplayProperties = shared.ListDisplayProperties{Key: true} }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, problems := domain.CheckPatch(tt.patch)
			if problems != nil {
				t.Fatal(problems)
			}
			want := stored()
			tt.want(&want)
			got, changed := p.Apply(s)
			if !reflect.DeepEqual(got, want) || changed != tt.changed {
				t.Fatalf("Apply = %s, %v; want %s, %v", show(got), changed, show(want), tt.changed)
			}
			if !reflect.DeepEqual(s, stored()) {
				t.Fatalf("Apply changed the content it was given: %s", show(s))
			}
		})
	}
}

// Two contents are equal when every field is, in value: pointers to equal values are equal, an explicitly empty
// filter is not an omitted one.
func TestContentEqual(t *testing.T) {
	if !stored().Equal(stored()) {
		t.Fatal("a content differs from its copy")
	}
	for _, tt := range []struct {
		name   string
		change func(*domain.Content)
	}{
		{"name", func(c *domain.Content) { c.Name = "mine" }},
		{"description", func(c *domain.Content) { c.Description += " " }},
		{"icon", func(c *domain.Content) { c.LogoProps.Icon.Color = nil }},
		{"an empty filter omitted", func(c *domain.Content) { c.Filters.LabelID = nil }},
		{"an omitted filter empty", func(c *domain.Content) { c.Filters.StateID = []string{} }},
		{"a filter value", func(c *domain.Content) { c.Filters.CycleID = []string{"d"} }},
		{"group", func(c *domain.Content) { c.DisplayFilters.GroupBy = ptr("labels") }},
		{"calendar", func(c *domain.Content) { c.DisplayFilters.Calendar.ShowWeekends = true }},
		{"a property", func(c *domain.Content) { c.DisplayProperties.Link = true }},
	} {
		changed := stored()
		tt.change(&changed)
		if stored().Equal(changed) || changed.Equal(stored()) {
			t.Errorf("%s: equal after a change", tt.name)
		}
	}
}
