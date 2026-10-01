package domain

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldsOf are the field errors of err, a 422 validation_failed, or nil.
func fieldsOf(t *testing.T, err error) []shared.FieldError {
	t.Helper()
	if err == nil {
		return nil
	}
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed {
		t.Fatalf("%v is not a validation_failed", err)
	}
	return e.Fields
}

// CheckProjectPatch accepts these and returns each as it is stored: the
// identifier in upper case, every other field as given, those not given
// nil; a lead or a default assignee cleared stays cleared.
func TestCheckProjectPatchAcceptsValidPatches(t *testing.T) {
	lead := uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	tests := []struct {
		p          ProjectPatch
		identifier *string
	}{
		{ProjectPatch{}, nil},
		{ProjectPatch{Identifier: ptr("çay1"), ArchiveIn: ptr(0)}, ptr("ÇAY1")},
		{ProjectPatch{Name: ptr("研发 Web"), Description: ptr("多行\n说明"), Network: ptr(NetworkPrivate), SetLead: true, LeadID: &lead,
			SetDefaultAssignee: true, CycleView: ptr(true), ModuleView: ptr(false), IssueViewsView: ptr(true), IntakeView: ptr(true),
			GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(12), LogoProps: &LogoProps{InUse: ptr(LogoEmoji)}, Timezone: ptr("Asia/Shanghai")}, nil},
	}
	for _, tt := range tests {
		got, err := CheckProjectPatch(tt.p)
		want := tt.p
		if tt.identifier != nil {
			want.Identifier = tt.identifier
		}
		if err != nil || !reflect.DeepEqual(got, want) {
			t.Errorf("CheckProjectPatch(%+v) = %+v, %v; want %+v", tt.p, got, err, want)
		}
	}
}

// archive_in takes 0 to 12 months, Plane's validators (M3 design 4.6); and
// every field with a problem is reported at once, in the order of the
// fields.
func TestCheckProjectPatchReportsEveryField(t *testing.T) {
	outOfRange := shared.FieldError{Field: "archive_in", Code: "out_of_range", Message: "must be between 0 and 12"}
	for _, months := range []int{-1, 13, 120} {
		if got := fieldsOf(t, func() error { _, err := CheckProjectPatch(ProjectPatch{ArchiveIn: &months}); return err }()); !slices.Equal(got,
			[]shared.FieldError{outOfRange}) {
			t.Errorf("archive_in %d: %+v, want %+v", months, got, outOfRange)
		}
	}
	const nul = "must not contain a NUL character"
	all := ProjectPatch{Name: ptr(""), Identifier: ptr("web-2"), Description: ptr("\x00"), Network: ptr(Network(1)), ArchiveIn: ptr(13),
		Timezone: ptr("Mars/Olympus"), LogoProps: &LogoProps{InUse: ptr("x")}}
	want := []shared.FieldError{
		{Field: "name", Code: "too_short", Message: "must not be empty"},
		{Field: "identifier", Code: "invalid_format", Message: "may hold only A-Z, 0-9 and ÇŞĞİÖÜ"},
		{Field: "description", Code: "invalid_format", Message: nul},
		{Field: "network", Code: "invalid_format", Message: "must be 0 (private) or 2 (public)"},
		outOfRange,
		{Field: "timezone", Code: "invalid_format", Message: "is not a known time zone"},
		{Field: "logo_props.in_use", Code: "invalid_format", Message: "must be emoji or icon"},
	}
	_, err := CheckProjectPatch(all)
	if got := fieldsOf(t, err); !slices.Equal(got, want) {
		t.Errorf("all at once: %+v, want %+v", got, want)
	}
}

// A field a patch gives is held to createProject's rule, problem for
// problem (M3 design 3.19): each value below, given to CheckNewProject in
// an otherwise valid project and to CheckProjectPatch alone, is refused
// with the same field errors, or accepted by both.
func TestThePatchChecksAsCreateDoes(t *testing.T) {
	valid := NewProject{Name: "Web", Identifier: "WEB"}
	type pair struct {
		create NewProject
		patch  ProjectPatch
	}
	var pairs []pair
	for _, name := range []string{"", " ", strings.Repeat("项", 255), strings.Repeat("项", 256), "W\x00", "Web-2", "Web.2", "研发 Web"} {
		create := valid
		create.Name = name
		pairs = append(pairs, pair{create, ProjectPatch{Name: &name}})
	}
	for _, id := range []string{"", "web", "çay1", "ABCDEFGHIJ", "ABCDEFGHIJK", "WEB-2", "WEB 2", "WEB\n", "ÉQUIPE"} {
		create := valid
		create.Identifier = id
		pairs = append(pairs, pair{create, ProjectPatch{Identifier: &id}})
	}
	for _, zone := range []string{"", "Local", "UTC", "Asia/Shanghai", "Mars/Olympus"} {
		create := valid
		create.Timezone = &zone
		pairs = append(pairs, pair{create, ProjectPatch{Timezone: &zone}})
	}
	for _, n := range []Network{0, 1, 2, 3} {
		create := valid
		create.Network = &n
		pairs = append(pairs, pair{create, ProjectPatch{Network: &n}})
	}
	for _, d := range []string{"", "a\x00", "多行\n说明"} {
		create := valid
		create.Description = d
		pairs = append(pairs, pair{create, ProjectPatch{Description: &d}})
	}
	for _, l := range []LogoProps{{}, {InUse: ptr("other")}, {Emoji: &Emoji{URL: ptr("\x00")}}, {Icon: &Icon{Name: ptr("home")}}} {
		create := valid
		create.LogoProps = l
		pairs = append(pairs, pair{create, ProjectPatch{LogoProps: &l}})
	}
	for _, p := range pairs {
		_, createErr := CheckNewProject(p.create)
		_, patchErr := CheckProjectPatch(p.patch)
		if c, u := fieldsOf(t, createErr), fieldsOf(t, patchErr); !slices.Equal(c, u) {
			t.Errorf("%+v: create %+v, update %+v; want the same", p.patch, c, u)
		}
	}
}

// Only a project's admins and members may be its lead or its default
// assignee on update, not its guests nor a role outside the three (M3
// design 3.19).
func TestCanAssign(t *testing.T) {
	for role, want := range map[shared.Role]bool{shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false,
		16: false, 25: false} {
		if got := CanAssign(role); got != want {
			t.Errorf("CanAssign(%d) = %v, want %v", role, got, want)
		}
	}
}
