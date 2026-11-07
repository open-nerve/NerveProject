package domain_test

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fieldsOf is the 422 validation_failed err carries, each problem as "field code"; nil, with a failure, for any other
// error.
func fieldsOf(t *testing.T, err error) []string {
	t.Helper()
	var e *shared.Error
	if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed {
		t.Errorf("%v; want 422 validation_failed", err)
		return nil
	}
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Field+" "+f.Code)
	}
	return out
}

func status(s string) *domain.ModuleStatus { v := domain.ModuleStatus(s); return &v }

// ids are n new accounts.
func ids(n int) []uuid.UUID {
	out := make([]uuid.UUID, n)
	for i := range out {
		out[i] = uuid.NewV7()
	}
	return out
}

// A creation's values are checked before anything is read (M6 P2 spec R2): the name cleaned (CleanName), a
// description with NUL, a status not one of the six as they are spelt, a start date after the target date (at
// target_date; each date may be given alone), more than 100 members or one named twice (at its index). Every problem
// at once, in the body's order. A status not given is planned; one given is kept.
func TestCheckNewModule(t *testing.T) {
	lead, a, b := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	got, err := domain.CheckNewModule(domain.NewModule{Name: "  Auth  ", Description: "Sign in", StartDate: day(t, "2026-11-07"),
		TargetDate: day(t, "2026-11-07"), LeadID: &lead, MemberIDs: []uuid.UUID{b, a}})
	want := domain.NewModule{Name: "Auth", Description: "Sign in", StartDate: day(t, "2026-11-07"), TargetDate: day(t, "2026-11-07"),
		Status: status("planned"), LeadID: &lead, MemberIDs: []uuid.UUID{b, a}}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("a clean creation: %+v, %v; want %+v (the same day for both dates is in order)", got, err, want)
	}
	for _, s := range []string{"backlog", "planned", "in-progress", "paused", "completed", "cancelled"} {
		if got, err := domain.CheckNewModule(domain.NewModule{Name: "M", Status: status(s)}); err != nil || *got.Status != domain.ModuleStatus(s) {
			t.Errorf("status %s: %v, %v", s, got.Status, err)
		}
	}
	for _, tt := range []struct {
		name string
		n    domain.NewModule
		want []string
	}{
		{"the start alone", domain.NewModule{Name: "M", StartDate: day(t, "2026-11-07")}, nil},
		{"the target alone", domain.NewModule{Name: "M", TargetDate: day(t, "2026-11-07")}, nil},
		{"a hundred members", domain.NewModule{Name: "M", MemberIDs: ids(100)}, nil},
		{"no member", domain.NewModule{Name: "M", MemberIDs: []uuid.UUID{}}, nil},
		{"a blank name", domain.NewModule{Name: " \t"}, []string{"name too_short"}},
		{"a long name", domain.NewModule{Name: strings.Repeat("模", 256)}, []string{"name too_long"}},
		{"a NUL description", domain.NewModule{Name: "M", Description: "a\x00b"}, []string{"description invalid_format"}},
		{"the start after the target", domain.NewModule{Name: "M", StartDate: day(t, "2026-11-08"), TargetDate: day(t, "2026-11-07")},
			[]string{"target_date not_allowed"}},
		{"an unknown status", domain.NewModule{Name: "M", Status: status("done")}, []string{"status invalid_format"}},
		{"a status in another case", domain.NewModule{Name: "M", Status: status("Planned")}, []string{"status invalid_format"}},
		{"a status spelt with _", domain.NewModule{Name: "M", Status: status("in_progress")}, []string{"status invalid_format"}},
		{"an empty status", domain.NewModule{Name: "M", Status: status("")}, []string{"status invalid_format"}},
		{"101 members", domain.NewModule{Name: "M", MemberIDs: ids(101)}, []string{"member_ids too_long"}},
		{"a member twice", domain.NewModule{Name: "M", MemberIDs: []uuid.UUID{a, b, a, a}}, []string{"member_ids[2] duplicate", "member_ids[3] duplicate"}},
		{"every problem", domain.NewModule{Name: "", Description: "\x00", StartDate: day(t, "2026-11-09"), TargetDate: day(t, "2026-11-08"),
			Status: status("x"), MemberIDs: []uuid.UUID{b, b}},
			[]string{"name too_short", "description invalid_format", "target_date not_allowed", "status invalid_format", "member_ids[1] duplicate"}},
	} {
		got, err := domain.CheckNewModule(tt.n)
		if tt.want == nil {
			if err != nil {
				t.Errorf("%s: refused, %v", tt.name, err)
			}
			continue
		}
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.NewModule{}) {
			t.Errorf("%s: %q, %+v; want %q", tt.name, fields, got, tt.want)
		}
	}
	_, err = domain.CheckNewModule(domain.NewModule{Name: "M", Status: status("done")})
	var e *shared.Error
	if errors.As(err, &e) && e.Fields[0].Message != "must be one of backlog, planned, in-progress, paused, completed, cancelled" {
		t.Errorf("the status' message %q names not the six", e.Fields[0].Message)
	}
}

// A module's lead and members are active members of its project who are not its guests (M6 P2 spec R5): an admin or
// a member may be; a guest, and an account who is no active member (no role), may not, each at its place. A creation
// makes them all anew, so each is held to it.
func TestNewModuleCheckRoles(t *testing.T) {
	admin, member, guest, gone := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	roles := map[uuid.UUID]shared.Role{admin: shared.RoleAdmin, member: shared.RoleMember, guest: shared.RoleGuest}
	for _, tt := range []struct {
		name    string
		lead    *uuid.UUID
		members []uuid.UUID
		want    []string
	}{
		{"none", nil, nil, nil},
		{"an admin and a member", &admin, []uuid.UUID{member, admin}, nil},
		{"a guest lead", &guest, []uuid.UUID{admin}, []string{"lead_id not_allowed"}},
		{"a lead who is no member", &gone, nil, []string{"lead_id not_allowed"}},
		{"a guest and one who is no member among the members", &member, []uuid.UUID{admin, guest, member, gone},
			[]string{"member_ids[1] not_allowed", "member_ids[3] not_allowed"}},
		{"every one", &guest, []uuid.UUID{gone}, []string{"lead_id not_allowed", "member_ids[0] not_allowed"}},
	} {
		err := domain.NewModule{Name: "M", LeadID: tt.lead, MemberIDs: tt.members}.CheckRoles(roles)
		switch {
		case tt.want == nil && err != nil:
			t.Errorf("%s: refused, %v", tt.name, err)
		case tt.want != nil && !reflect.DeepEqual(fieldsOf(t, err), tt.want):
			t.Errorf("%s: %v; want %q", tt.name, err, tt.want)
		}
	}
	err := domain.NewModule{Name: "M", LeadID: &guest}.CheckRoles(roles)
	var e *shared.Error
	if errors.As(err, &e) && e.Fields[0].Message != "must be an active member of the project who is not its guest" {
		t.Errorf("the message %q", e.Fields[0].Message)
	}
}

// The module's refusals: their kinds and codes, as its contract declares them (M6 P2 spec 2.3, R13).
func TestModuleErrors(t *testing.T) {
	for _, tt := range []struct {
		err  *shared.Error
		kind shared.Kind
		code string
	}{
		{domain.ErrModuleNotFound, shared.KindNotFound, "planning.module_not_found"},
		{domain.ErrModulesDisabled, shared.KindConflict, "planning.modules_disabled"},
		{domain.ErrModuleNameTaken, shared.KindConflict, "planning.module_name_taken"},
		{domain.ErrModuleNotClosed, shared.KindConflict, "planning.module_not_closed"},
		{domain.ErrModuleLinkNotFound, shared.KindNotFound, "planning.module_link_not_found"},
		{domain.ErrModuleLinkExists, shared.KindConflict, "planning.module_link_exists"},
		{domain.ErrArchived, shared.KindConflict, "planning.archived"},
	} {
		if tt.err.Kind != tt.kind || tt.err.Code != tt.code {
			t.Errorf("%s: %v %s; want %v %s", tt.code, tt.err.Kind, tt.err.Code, tt.kind, tt.code)
		}
	}
	// planning.archived answers a cycle's change and a module's alike (R13).
	if got := domain.ErrArchived.Detail; got != "The cycle or module is archived; unarchive it to change it." {
		t.Errorf("planning.archived says %q", got)
	}
}

// The modules' actions are the three module.*: a write of one is held to the project's module toggle, a cycle
// action's to its cycle toggle (M6 design 3.8).
func TestModuleAction(t *testing.T) {
	for _, a := range domain.Actions() {
		if got, want := domain.ModuleAction(a), strings.HasPrefix(string(a), "module."); got != want {
			t.Errorf("ModuleAction(%s) = %t, want %t", a, got, want)
		}
	}
}
