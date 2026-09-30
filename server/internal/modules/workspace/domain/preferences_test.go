package domain

import (
	"errors"
	"math"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The fields a patch sets change, the others stay.
func TestPreferencesApply(t *testing.T) {
	base := Preferences{NavigationControl: "TABBED", NavigationProjectLimit: 3}
	tests := []struct {
		p    PreferencesPatch
		want Preferences
	}{
		{PreferencesPatch{}, base},
		{PreferencesPatch{NavigationControl: ptr("ACCORDION")}, Preferences{"ACCORDION", 3}},
		{PreferencesPatch{NavigationProjectLimit: ptr(0)}, Preferences{"TABBED", 0}},
		{PreferencesPatch{NavigationControl: ptr("ACCORDION"), NavigationProjectLimit: ptr(20)}, Preferences{"ACCORDION", 20}},
	}
	for _, tt := range tests {
		if got := base.Apply(tt.p); got != tt.want {
			t.Errorf("Apply(%+v) = %+v, want %+v", tt.p, got, tt.want)
		}
	}
	if DefaultPreferences() != (Preferences{"ACCORDION", 10}) {
		t.Errorf("DefaultPreferences() = %+v, want ACCORDION, 10", DefaultPreferences())
	}
}

// A patch has a mode of the two and a limit from 0 to 2147483647, the
// column's range; each field is checked only when set.
func TestCheckPreferencesPatch(t *testing.T) {
	for _, p := range []PreferencesPatch{
		{}, {NavigationControl: ptr("ACCORDION")}, {NavigationControl: ptr("TABBED")},
		{NavigationProjectLimit: ptr(0)}, {NavigationProjectLimit: ptr(math.MaxInt32)},
	} {
		if err := CheckPreferencesPatch(p); err != nil {
			t.Errorf("CheckPreferencesPatch(%+v) = %v, want nil", p, err)
		}
	}
	mode := shared.FieldError{Field: "navigation_control_preference", Code: "invalid_format", Message: "is not ACCORDION or TABBED"}
	limit := shared.FieldError{Field: "navigation_project_limit", Code: "out_of_range", Message: "must be between 0 and 2147483647"}
	tests := []struct {
		name string
		p    PreferencesPatch
		want []shared.FieldError
	}{
		{"a lower-case mode", PreferencesPatch{NavigationControl: ptr("tabbed")}, []shared.FieldError{mode}},
		{"a mode of Plane's sidebar", PreferencesPatch{NavigationControl: ptr("SIDEBAR")}, []shared.FieldError{mode}},
		{"no mode", PreferencesPatch{NavigationControl: ptr("")}, []shared.FieldError{mode}},
		{"a negative limit", PreferencesPatch{NavigationProjectLimit: ptr(-1)}, []shared.FieldError{limit}},
		{"a limit past the column", PreferencesPatch{NavigationProjectLimit: ptr(math.MaxInt32 + 1)}, []shared.FieldError{limit}},
		{"both", PreferencesPatch{NavigationControl: ptr("TABS"), NavigationProjectLimit: ptr(-5)}, []shared.FieldError{mode, limit}},
	}
	for _, tt := range tests {
		err := CheckPreferencesPatch(tt.p)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, tt.want) {
			t.Errorf("%s: CheckPreferencesPatch(%+v) = %#v, want validation_failed with %v", tt.name, tt.p, err, tt.want)
		}
	}
}
