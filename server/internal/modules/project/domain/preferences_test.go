package domain

import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SortOrderFirst puts a new project before every other of the member's in
// the workspace, or at the default when he has none (M3 design 3.18): 10000
// before the least of his places, though it be past the default.
func TestSortOrderFirst(t *testing.T) {
	for _, tt := range []struct {
		lowest *float64
		want   float64
	}{{nil, 65535}, {ptr(65535.0), 55535}, {ptr(-2.5), -10002.5}, {ptr(0.0), -10000}, {ptr(70000.0), 60000}} {
		if got := SortOrderFirst(tt.lowest); got != tt.want {
			t.Errorf("SortOrderFirst(%v) = %v, want %v", tt.lowest, got, tt.want)
		}
	}
}

// The defaults are the work items' tab, nothing hidden, the default place;
// a patch changes what it gives, the navigation whole, and nothing else.
func TestPreferencesApply(t *testing.T) {
	d := DefaultPreferences()
	if want := (Preferences{Navigation: Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}, SortOrder: 65535}); !reflect.DeepEqual(d, want) {
		t.Errorf("DefaultPreferences() = %+v, want %+v", d, want)
	}
	tabbed := Navigation{DefaultTab: "modules", HideInMoreMenu: []string{"views"}}
	for _, tt := range []struct {
		patch PreferencesPatch
		want  Preferences
	}{
		{PreferencesPatch{}, d},
		{PreferencesPatch{Navigation: &tabbed}, Preferences{Navigation: tabbed, SortOrder: 65535}},
		{PreferencesPatch{SortOrder: ptr(-1.5)}, Preferences{Navigation: d.Navigation, SortOrder: -1.5}},
	} {
		if got := d.Apply(tt.patch); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Apply(%+v) = %+v, want %+v", tt.patch, got, tt.want)
		}
	}
}

// Every tab is a default tab, and every one but the work items can be
// hidden, each once, in any order; no navigation is checked when none is
// given, whatever the place.
func TestCheckPreferencesPatchAcceptsEveryTab(t *testing.T) {
	for _, p := range []PreferencesPatch{
		{},
		{SortOrder: ptr(-1e9)},
		{Navigation: &Navigation{DefaultTab: "work_items", HideInMoreMenu: []string{}}},
		{Navigation: &Navigation{DefaultTab: "cycles", HideInMoreMenu: []string{"intake", "views", "modules", "cycles"}}},
		{Navigation: &Navigation{DefaultTab: "modules"}},
		{Navigation: &Navigation{DefaultTab: "views"}},
		{Navigation: &Navigation{DefaultTab: "intake"}},
	} {
		if err := CheckPreferencesPatch(p); err != nil {
			t.Errorf("CheckPreferencesPatch(%+v) = %v, want nil", p, err)
		}
	}
}

// An unknown tab as the default, or hidden, the work items hidden, a tab
// hidden twice: each refused, all of them in one 422 that names each by its
// place in the request.
func TestCheckPreferencesPatchReportsEveryTab(t *testing.T) {
	err := CheckPreferencesPatch(PreferencesPatch{Navigation: &Navigation{DefaultTab: "pages",
		HideInMoreMenu: []string{"views", "work_items", "pages", "views", "Views"}}})
	tabs, hideable := "is not one of work_items, cycles, modules, views, intake", "is not one of cycles, modules, views, intake"
	want := []shared.FieldError{
		{Field: "navigation.default_tab", Code: "invalid_format", Message: tabs},
		{Field: "navigation.hide_in_more_menu[1]", Code: "invalid_format", Message: hideable},
		{Field: "navigation.hide_in_more_menu[2]", Code: "invalid_format", Message: hideable},
		{Field: "navigation.hide_in_more_menu[3]", Code: "duplicate", Message: "is listed before"},
		{Field: "navigation.hide_in_more_menu[4]", Code: "invalid_format", Message: hideable},
	}
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
		t.Errorf("CheckPreferencesPatch() = %v; want validation_failed with\n%+v", err, want)
	}
}
