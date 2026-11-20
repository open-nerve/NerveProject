package domain_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// fields are the field and code of each problem, in order.
func fields(problems []shared.FieldError) []string {
	var out []string
	for _, p := range problems {
		out = append(out, p.Field+" "+p.Code)
	}
	return out
}

// A creation's content (P3b spec 2.4, DP-22): the name without its surrounding white space, the description as given,
// no icon and no filter when it gives none, the shared display defaults for the display settings it does not give. The
// given ones are kept as given, an explicitly empty filter apart from an omitted one.
func TestCheckNewViewFillsTheDefaults(t *testing.T) {
	got, problems := domain.CheckNewView(domain.NewView{Name: " \tMine\n", Description: "  as typed  "})
	want := domain.Content{Name: "Mine", Description: "  as typed  ",
		DisplayFilters: shared.DefaultListDisplayFilters(), DisplayProperties: shared.DefaultListDisplayProperties()}
	if problems != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("CheckNewView = %+v, %v; want %+v", got, problems, want)
	}
	display := shared.ListDisplayFilters{Layout: "spreadsheet", OrderBy: "priority", Calendar: shared.ListCalendar{Layout: "week", ShowWeekends: true}}
	properties := shared.ListDisplayProperties{Key: true}
	filters := domain.Filters{ListFilters: shared.ListFilters{Priority: []string{"urgent"}, LabelID: []string{}}, ProjectID: []string{"p"}}
	logo := shared.LogoProps{InUse: ptr(shared.LogoInUseIcon), Icon: &shared.LogoIcon{Name: ptr("star")}}
	got, problems = domain.CheckNewView(domain.NewView{Name: "Triage", LogoProps: logo, Filters: filters, DisplayFilters: &display, DisplayProperties: &properties})
	want = domain.Content{Name: "Triage", LogoProps: logo, Filters: filters, DisplayFilters: display, DisplayProperties: properties}
	if problems != nil || !reflect.DeepEqual(got, want) || got.Filters.LabelID == nil || got.Filters.StateID != nil {
		t.Fatalf("CheckNewView = %+v, %v; want %+v, the empty label filter kept apart from the omitted state", got, problems, want)
	}
}

// A view's name is trimmed and then 1 to 255 characters, counted as characters, without NUL (spec 2.3; the column's
// CHECK holds only that it is not empty). A blank name is too short; Plane's views have no other name rule.
func TestAViewsName(t *testing.T) {
	long := strings.Repeat("视", 255)
	for _, name := range []string{long, " " + long + "\n"} {
		if got, problems := domain.CheckNewView(domain.NewView{Name: name}); problems != nil || got.Name != long {
			t.Fatalf("255 characters: %q, %v", got.Name, problems)
		}
	}
	for _, tt := range []struct{ name, in, code string }{
		{"empty", "", shared.FieldTooShort},
		{"blank", " \t\n　", shared.FieldTooShort},
		{"256 characters", strings.Repeat("视", 256), shared.FieldTooLong},
		{"NUL", "Mi\x00ne", shared.FieldInvalidFormat},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, problems := domain.CheckNewView(domain.NewView{Name: tt.in})
			if got := fields(problems); len(got) != 1 || got[0] != "name "+tt.code {
				t.Fatalf("create: %v; want name %s", got, tt.code)
			}
			_, problems = domain.CheckPatch(domain.Patch{Name: &tt.in})
			if got := fields(problems); len(got) != 1 || got[0] != "name "+tt.code {
				t.Fatalf("patch: %v; want name %s", got, tt.code)
			}
		})
	}
}

// Every problem of a creation's or a change's name, description and icon is reported at once, in the body's order:
// the use case answers them in one 422, the words of the view's kind after them (app). An icon of another in_use,
// or NUL in any of its texts or in the description, is refused; the description is plain text, of any length.
func TestEveryProblemAtOnceInTheBodysOrder(t *testing.T) {
	bad := shared.LogoProps{InUse: ptr("svg"), Emoji: &shared.LogoEmoji{Value: ptr("\x00")}, Icon: &shared.LogoIcon{Color: ptr("#f\x00")}}
	want := []string{"name too_short", "description invalid_format", "logo_props.in_use invalid_format",
		"logo_props.emoji.value invalid_format", "logo_props.icon.color invalid_format"}
	_, problems := domain.CheckNewView(domain.NewView{Name: " ", Description: "a\x00b", LogoProps: bad})
	if got := fields(problems); !reflect.DeepEqual(got, want) {
		t.Fatalf("create: %v; want %v", got, want)
	}
	_, problems = domain.CheckPatch(domain.Patch{Name: ptr(" "), Description: ptr("a\x00b"), LogoProps: &bad})
	if got := fields(problems); !reflect.DeepEqual(got, want) {
		t.Fatalf("patch: %v; want %v", got, want)
	}
	for _, logo := range []shared.LogoProps{{InUse: ptr("svg")}, {InUse: ptr("")}} {
		if _, problems := domain.CheckNewView(domain.NewView{Name: "Mine", LogoProps: logo}); len(problems) != 1 {
			t.Fatalf("in_use %q accepted: %v", *logo.InUse, problems)
		}
	}
	long := strings.Repeat("说明", 10000)
	if got, problems := domain.CheckNewView(domain.NewView{Name: "Mine", Description: long}); problems != nil || got.Description != long {
		t.Fatalf("a long description: %v", problems)
	}
}

// A change checks only the fields it gives, and returns them as stored: its name trimmed, the rest as given. {} is
// a change with nothing to check.
func TestCheckPatch(t *testing.T) {
	if got, problems := domain.CheckPatch(domain.Patch{}); problems != nil || !reflect.DeepEqual(got, domain.Patch{}) {
		t.Fatalf("{} = %+v, %v", got, problems)
	}
	logo := shared.LogoProps{InUse: ptr(shared.LogoInUseEmoji), Emoji: &shared.LogoEmoji{Value: ptr("128512")}}
	p := domain.Patch{Name: ptr("  Renamed "), Description: ptr(" kept "), LogoProps: &logo, Filters: &domain.Filters{CycleID: []string{}}}
	got, problems := domain.CheckPatch(p)
	if problems != nil || *got.Name != "Renamed" || *got.Description != " kept " || !reflect.DeepEqual(got.LogoProps, &logo) ||
		got.Filters.CycleID == nil || got.DisplayFilters != nil || got.DisplayProperties != nil {
		t.Fatalf("CheckPatch = %+v, %v", got, problems)
	}
}

// A view's filters are the contract's union of thirteen keys (spec 2.4): the ten every saved list has, a project
// view's cycle_id and module_id, a workspace view's project_id; which of them a view may hold is its kind's words. As
// JSON and as the map the settings port takes, a nil slice is an omitted key and an empty one a key kept.
func TestFiltersKeepOmittedApartFromEmpty(t *testing.T) {
	f := domain.Filters{ListFilters: shared.ListFilters{StateID: []string{"s"}, TargetDate: []string{}},
		CycleID: []string{"c"}, ModuleID: []string{}, ProjectID: []string{"p"}}
	want := map[string][]string{"state_id": {"s"}, "target_date": {}, "cycle_id": {"c"}, "module_id": {}, "project_id": {"p"}}
	if got := f.Map(); !reflect.DeepEqual(got, want) || got["target_date"] == nil || got["module_id"] == nil {
		t.Fatalf("Map = %#v; want %#v", got, want)
	}
	if got := (domain.Filters{}).Map(); got == nil || len(got) != 0 {
		t.Fatalf("no keys: %#v; want an empty map", got)
	}
	data, err := json.Marshal(f)
	if wantJSON := `{"state_id":["s"],"target_date":[],"cycle_id":["c"],"module_id":[],"project_id":["p"]}`; err != nil || string(data) != wantJSON {
		t.Fatalf("JSON = %s, %v; want %s", data, err, wantJSON)
	}
	var back domain.Filters
	if err := json.Unmarshal(data, &back); err != nil || !reflect.DeepEqual(back, f) {
		t.Fatalf("round trip = %+v, %v", back, err)
	}
	if data, err := json.Marshal(domain.Filters{}); err != nil || string(data) != "{}" {
		t.Fatalf("no filter = %s, %v; want {}", data, err)
	}
}
