package domain_test

import (
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// Defaults and whole-field replacement are independent of storage and HTTP.
func TestProjectIssuePreferencesValues(t *testing.T) {
	want := domain.Preferences{
		Navigation: domain.Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}, SortOrder: 65535,
		Filters:           domain.Filters{},
		DisplayFilters:    domain.DisplayFilters{Layout: "list", OrderBy: "-created_at", SubIssue: true, ShowEmptyGroups: true, Calendar: domain.DisplayCalendar{Layout: "month"}},
		DisplayProperties: domain.DisplayProperties{Assignee: true, StartDate: true, DueDate: true, Labels: true, Key: true, Priority: true, State: true, SubIssueCount: true, Link: true, CreatedOn: true, UpdatedOn: true},
	}
	p := domain.DefaultPreferences()
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("defaults = %+v, want %+v", p, want)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{LabelID: []string{"none"}, Priority: []string{"high"}}})
	want.Filters = domain.Filters{LabelID: []string{"none"}, Priority: []string{"high"}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("first filters = %+v", p)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{LabelID: []string{}}})
	want.Filters = domain.Filters{LabelID: []string{}}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("replacement/empty array = %+v", p)
	}
	display := domain.DisplayFilters{Layout: "kanban", OrderBy: "sort_order", Calendar: domain.DisplayCalendar{Layout: "week"}}
	p = p.Apply(domain.PreferencesPatch{DisplayFilters: &display, DisplayProperties: &domain.DisplayProperties{Key: true}})
	want.DisplayFilters = display
	want.DisplayProperties = domain.DisplayProperties{Key: true}
	if !reflect.DeepEqual(p, want) {
		t.Fatalf("false flags and omitted filters = %+v", p)
	}
	p = p.Apply(domain.PreferencesPatch{Filters: &domain.Filters{}})
	want.Filters = domain.Filters{}
	if !reflect.DeepEqual(p, want) || !reflect.DeepEqual(p.Apply(domain.PreferencesPatch{}), want) {
		t.Fatalf("clear/omit = %+v", p)
	}
}
