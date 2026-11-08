package domain

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A patch's modules (M6 P3a spec 2.3): not given changes nothing; given,
// they are the work item's new set, whose difference with its own is added
// and removed, each in id order, whatever the order given; the same set in
// another order changes nothing, and Changes.None knows it; given empty,
// every module is removed.
func TestChangesOfTheModules(t *testing.T) {
	ids := slices.SortedFunc(slices.Values([]uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}), uuid.UUID.Compare)
	a, b, c, d := ids[0], ids[1], ids[2], ids[3]
	i := Issue{ModuleIDs: []uuid.UUID{a, c}}
	for _, tt := range []struct {
		name string
		p    IssuePatch
		want Diff
	}{
		{"not given", IssuePatch{}, Diff{}},
		{"the same, in another order", IssuePatch{ModuleIDs: &[]uuid.UUID{c, a}}, Diff{}},
		{"replaced", IssuePatch{ModuleIDs: &[]uuid.UUID{d, c, b}}, Diff{Added: []uuid.UUID{b, d}, Removed: []uuid.UUID{a}}},
		{"emptied", IssuePatch{ModuleIDs: &[]uuid.UUID{}}, Diff{Removed: []uuid.UUID{a, c}}},
		{"one more", IssuePatch{ModuleIDs: &[]uuid.UUID{d, a, c}}, Diff{Added: []uuid.UUID{d}}},
	} {
		got := ChangesOf(i, tt.p)
		if !reflect.DeepEqual(got.Modules, tt.want) || got.None() != (len(tt.want.Added)+len(tt.want.Removed) == 0) {
			t.Errorf("%s: ChangesOf().Modules = %+v, None() %t; want %+v", tt.name, got.Modules, got.None(), tt.want)
		}
	}
}

// The rows of a work item's modules (M6 P3a spec 2.4, Plane's field and
// verbs): each module joined is created, from "" to its name, its id the
// new identifier; each left is deleted, from its name to "", its id the old
// identifier; Plane's sentences go with them, without the names. An update
// writes them after its cycle's row, those joined then those left, each in
// the order given (UpdateActivity); a creation writes the modules it joins
// after the cycle it joins (CreationJoins).
func TestModuleActivity(t *testing.T) {
	auth, billing := Named{ID: uuid.NewV7(), Name: "Auth"}, Named{ID: uuid.NewV7(), Name: "Billing"}
	sprint := Named{ID: uuid.NewV7(), Name: "Sprint"}
	field, empty := "modules", ""
	joined := func(m Named) Activity {
		return Activity{Field: &field, Verb: VerbCreated, OldValue: &empty, NewValue: &m.Name, NewIdentifier: &m.ID, Comment: "added module "}
	}
	left := func(m Named) Activity {
		return Activity{Field: &field, Verb: VerbDeleted, OldValue: &m.Name, NewValue: &empty, OldIdentifier: &m.ID,
			Comment: "removed this work item from module "}
	}
	cycle := UpdateActivity(Update{Cycle: &Changed[*Named]{New: &sprint}})[0]
	rows := UpdateActivity(Update{Name: &Changed[string]{Old: "Fix", New: "Mend"}, Cycle: &Changed[*Named]{New: &sprint},
		ModulesAdded: []Named{auth, billing}, ModulesRemoved: []Named{billing, auth}})
	want := []Activity{UpdateActivity(Update{Name: &Changed[string]{Old: "Fix", New: "Mend"}})[0], cycle, joined(auth), joined(billing),
		left(billing), left(auth)}
	if !reflect.DeepEqual(rows, want) {
		t.Errorf("UpdateActivity() = %+v\nwant %+v", rows, want)
	}
	if got := CreationJoins(&sprint, auth, billing); !reflect.DeepEqual(got, []Activity{cycle, joined(auth), joined(billing)}) {
		t.Errorf("CreationJoins(Sprint, Auth, Billing) = %+v; want the cycle's, then each module's", got)
	}
	if got := CreationJoins(nil, billing); !reflect.DeepEqual(got, []Activity{joined(billing)}) {
		t.Errorf("CreationJoins(none, Billing) = %+v; want the module's", got)
	}
	if got := CreationJoins(nil); got != nil {
		t.Errorf("CreationJoins(none) = %+v; want none", got)
	}
}

// module_ids is checked as label_ids (checkIDs, M6 P3a spec 2.3), after
// every other field, on a creation and a patch alike: at most MaxModules,
// a list too long reported alone as its one problem, its ids not looked
// at; otherwise each id named before, at its index. A hundred, and none,
// are taken as given.
func TestModulesAreCheckedAsLabels(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	hundred, many := ids(MaxModules), append(ids(MaxModules), a, a)
	tooLong := shared.FieldError{Field: "module_ids", Code: "too_long", Message: "must name at most 100"}
	repeats := []shared.FieldError{{Field: "module_ids[2]", Code: "duplicate", Message: "is listed before"},
		{Field: "module_ids[3]", Code: "duplicate", Message: "is listed before"}}
	name := shared.FieldError{Field: "name", Code: "too_short", Message: "must not be empty"}
	for _, tt := range []struct {
		name    string
		modules []uuid.UUID
		want    []shared.FieldError
	}{
		{"too many, with repeats", many, []shared.FieldError{name, tooLong}},
		{"repeats", []uuid.UUID{a, b, a, b}, append([]shared.FieldError{name}, repeats...)},
	} {
		var e *shared.Error
		if _, err := CheckNewIssue(NewIssue{Name: " ", ModuleIDs: tt.modules}); !errors.As(err, &e) || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckNewIssue() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
		if _, err := CheckIssuePatch(IssuePatch{Name: ptr(""), ModuleIDs: &tt.modules}); !errors.As(err, &e) || !reflect.DeepEqual(e.Fields, tt.want) {
			t.Errorf("%s: CheckIssuePatch() = %v; want validation_failed with %+v", tt.name, err, tt.want)
		}
	}
	for _, given := range [][]uuid.UUID{hundred, {}} {
		if got, err := CheckNewIssue(NewIssue{Name: "Fix", ModuleIDs: given}); err != nil || !slices.Equal(got.ModuleIDs, given) {
			t.Errorf("CheckNewIssue(%d modules) = %+v, %v; want them as given", len(given), got.ModuleIDs, err)
		}
		if got, err := CheckIssuePatch(IssuePatch{ModuleIDs: &given}); err != nil || got.ModuleIDs == nil || !slices.Equal(*got.ModuleIDs, given) {
			t.Errorf("CheckIssuePatch(%d modules) = %+v, %v; want them as given", len(given), got.ModuleIDs, err)
		}
	}
}

// skip_activity goes with description_html alone, so not with the modules,
// given or emptied, which the description's own save would not write.
func TestSkipActivityRefusesTheModules(t *testing.T) {
	html, skip := "<p>Fix</p>", true
	for _, modules := range [][]uuid.UUID{{uuid.NewV7()}, {}} {
		p := IssuePatch{DescriptionHTML: &html, SkipActivity: &skip, ModuleIDs: &modules}
		_, err := CheckIssuePatch(p)
		var e *shared.Error
		if !errors.As(err, &e) || len(e.Fields) != 1 || e.Fields[0].Field != "skip_activity" {
			t.Errorf("CheckIssuePatch(%d modules with skip_activity) = %v; want skip_activity refused", len(modules), err)
		}
	}
}

// A module that a write may not join is the 422's problem at its index in
// module_ids; turned off, the modules refuse a change with planning's
// code.
func TestModuleNotAllowed(t *testing.T) {
	want := shared.FieldError{Field: "module_ids[3]", Code: shared.FieldNotAllowed, Message: "must be a module of the project"}
	if got := ModuleNotAllowed(3); got != want {
		t.Errorf("ModuleNotAllowed(3) = %+v, want %+v", got, want)
	}
	var e *shared.Error
	if !errors.As(ErrModulesDisabled, &e) || e.Kind != shared.KindConflict || e.Code != "planning.modules_disabled" ||
		e.Detail != "Modules are turned off in this project; turn them on to change a work item's modules." {
		t.Errorf("ErrModulesDisabled = %+v; want 409 planning.modules_disabled with the work item's detail", ErrModulesDisabled)
	}
}
