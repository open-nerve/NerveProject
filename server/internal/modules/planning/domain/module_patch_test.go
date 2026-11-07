package domain_test

import (
	"math"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ascending are n new accounts in id order.
func ascending(n int) []uuid.UUID {
	out := ids(n)
	slices.SortFunc(out, uuid.UUID.Compare)
	return out
}

// A change's values are checked before anything is read (M6 P2 spec R2): those it gives alone — the name cleaned, a
// description with NUL, a status not one of the six, more than 100 members or one named twice, a sort order that is no
// finite number; every problem at once, in the body's order. Its dates and its lead and members are held to the module
// under the lock (Apply), not here.
func TestCheckModulePatch(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	in := domain.ModulePatch{Name: ptr("  Auth "), Status: status("paused"), MemberIDs: &[]uuid.UUID{b, a}, SortOrder: ptr(-1.5),
		SetStartDate: true, StartDate: day(t, "2026-11-09"), SetTargetDate: true, TargetDate: day(t, "2026-11-08"), SetLeadID: true}
	got, err := domain.CheckModulePatch(in)
	want := in
	want.Name = ptr("Auth")
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("a clean change: %+v, %v; want %+v", got, err, want)
	}
	if got, err := domain.CheckModulePatch(domain.ModulePatch{}); err != nil || !reflect.DeepEqual(got, domain.ModulePatch{}) {
		t.Fatalf("an empty change: %+v, %v", got, err)
	}
	for _, tt := range []struct {
		name string
		p    domain.ModulePatch
		want []string
	}{
		{"a blank name", domain.ModulePatch{Name: ptr(" ")}, []string{"name too_short"}},
		{"a long name", domain.ModulePatch{Name: ptr(strings.Repeat("x", 256))}, []string{"name too_long"}},
		{"a NUL description", domain.ModulePatch{Description: ptr("\x00")}, []string{"description invalid_format"}},
		{"an unknown status", domain.ModulePatch{Status: status("In-Progress")}, []string{"status invalid_format"}},
		{"101 members", domain.ModulePatch{MemberIDs: ptr(ids(101))}, []string{"member_ids too_long"}},
		{"a member twice", domain.ModulePatch{MemberIDs: &[]uuid.UUID{a, a}}, []string{"member_ids[1] duplicate"}},
		{"NaN", domain.ModulePatch{SortOrder: ptr(math.NaN())}, []string{"sort_order invalid_format"}},
		{"infinity", domain.ModulePatch{SortOrder: ptr(math.Inf(-1))}, []string{"sort_order invalid_format"}},
		{"every problem", domain.ModulePatch{Name: ptr(""), Description: ptr("\x00"), Status: status(""), MemberIDs: &[]uuid.UUID{b, b},
			SortOrder: ptr(math.Inf(1))},
			[]string{"name too_short", "description invalid_format", "status invalid_format", "member_ids[1] duplicate", "sort_order invalid_format"}},
	} {
		got, err := domain.CheckModulePatch(tt.p)
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) || !reflect.DeepEqual(got, domain.ModulePatch{}) {
			t.Errorf("%s: %q, %+v; want %q", tt.name, fields, got, tt.want)
		}
	}
}

// Apply: the module as the change leaves it, and whether it changes anything. A field given as it is changes nothing,
// and neither do member_ids naming the module's members in another order (M6 P2 spec R4); the members it leaves are
// in id order. A date and the lead may be cleared; each date may be given alone, the module may keep only one.
func TestModulePatchApply(t *testing.T) {
	people := ascending(4)
	a, b, c, lead := people[0], people[1], people[2], people[3]
	roles := map[uuid.UUID]shared.Role{a: shared.RoleMember, b: shared.RoleMember, c: shared.RoleAdmin, lead: shared.RoleAdmin}
	m := domain.Module{Name: "Auth", Description: "Sign in", StartDate: day(t, "2026-11-02"), TargetDate: day(t, "2026-11-13"),
		Status: domain.ModulePlanned, LeadID: &lead, MemberIDs: []uuid.UUID{a, b}, SortOrder: 3}
	undated := domain.Module{Name: "Draft", Status: domain.ModuleBacklog, SortOrder: 3}
	with := func(base domain.Module, change func(*domain.Module)) domain.Module {
		change(&base)
		return base
	}
	for _, tt := range []struct {
		name    string
		m       domain.Module
		p       domain.ModulePatch
		want    domain.Module
		changed bool
	}{
		{"nothing", m, domain.ModulePatch{}, m, false},
		{"each as it is", m, domain.ModulePatch{Name: ptr("Auth"), Description: ptr("Sign in"), SetStartDate: true, StartDate: day(t, "2026-11-02"),
			SetTargetDate: true, TargetDate: day(t, "2026-11-13"), Status: status("planned"), SetLeadID: true, LeadID: &lead,
			MemberIDs: &[]uuid.UUID{b, a}, SortOrder: ptr(3.0)}, m, false},
		{"each changed", m, domain.ModulePatch{Name: ptr("Two"), Description: ptr(""), SetStartDate: true, StartDate: day(t, "2026-11-01"),
			SetTargetDate: true, TargetDate: day(t, "2026-11-01"), Status: status("in-progress"), SetLeadID: true, LeadID: &c,
			MemberIDs: &[]uuid.UUID{c, a}, SortOrder: ptr(-7.5)},
			domain.Module{Name: "Two", StartDate: day(t, "2026-11-01"), TargetDate: day(t, "2026-11-01"), Status: domain.ModuleInProgress,
				LeadID: &c, MemberIDs: []uuid.UUID{a, c}, SortOrder: -7.5}, true},
		{"the name alone", m, domain.ModulePatch{Name: ptr("auth")}, with(m, func(n *domain.Module) { n.Name = "auth" }), true},
		{"the description alone", m, domain.ModulePatch{Description: ptr("")}, with(m, func(n *domain.Module) { n.Description = "" }), true},
		{"the status alone", m, domain.ModulePatch{Status: status("cancelled")}, with(m, func(n *domain.Module) { n.Status = domain.ModuleCancelled }), true},
		{"the sort order alone", m, domain.ModulePatch{SortOrder: ptr(4.0)}, with(m, func(n *domain.Module) { n.SortOrder = 4 }), true},
		{"the lead cleared", m, domain.ModulePatch{SetLeadID: true}, with(m, func(n *domain.Module) { n.LeadID = nil }), true},
		{"a lead given", undated, domain.ModulePatch{SetLeadID: true, LeadID: &a}, with(undated, func(n *domain.Module) { n.LeadID = &a }), true},
		{"the start alone", m, domain.ModulePatch{SetStartDate: true, StartDate: day(t, "2026-11-13")},
			with(m, func(n *domain.Module) { n.StartDate = day(t, "2026-11-13") }), true},
		{"the target alone", m, domain.ModulePatch{SetTargetDate: true, TargetDate: day(t, "2026-11-30")},
			with(m, func(n *domain.Module) { n.TargetDate = day(t, "2026-11-30") }), true},
		{"the start cleared alone", m, domain.ModulePatch{SetStartDate: true}, with(m, func(n *domain.Module) { n.StartDate = nil }), true},
		{"the target cleared alone", m, domain.ModulePatch{SetTargetDate: true}, with(m, func(n *domain.Module) { n.TargetDate = nil }), true},
		{"an undated module's target alone", undated, domain.ModulePatch{SetTargetDate: true, TargetDate: day(t, "2026-11-30")},
			with(undated, func(n *domain.Module) { n.TargetDate = day(t, "2026-11-30") }), true},
		{"a member added", m, domain.ModulePatch{MemberIDs: &[]uuid.UUID{c, b, a}}, with(m, func(n *domain.Module) { n.MemberIDs = []uuid.UUID{a, b, c} }), true},
		{"a member removed", m, domain.ModulePatch{MemberIDs: &[]uuid.UUID{b}}, with(m, func(n *domain.Module) { n.MemberIDs = []uuid.UUID{b} }), true},
		{"every member removed", m, domain.ModulePatch{MemberIDs: &[]uuid.UUID{}}, with(m, func(n *domain.Module) { n.MemberIDs = nil }), true},
		{"no member, as it has", undated, domain.ModulePatch{MemberIDs: &[]uuid.UUID{}}, undated, false},
	} {
		got, changed, err := tt.p.Apply(tt.m, roles)
		if err != nil || changed != tt.changed || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v, %t, %v; want %+v, %t", tt.name, got, changed, err, tt.want, tt.changed)
		}
	}
}

// Apply refuses under the lock, every problem at once, each at its place in the change: the dates it gives, each else
// the module's, with the start after the target (M6 P2 spec R4) — at start_date when it gives the start alone, else at
// target_date; the lead it sets anew and each member it adds who may not be (R5). A lead or a member the module keeps
// is not looked at, even one who has left the project since; a lead set anew is, though he is a member it keeps.
func TestModulePatchApplyRefuses(t *testing.T) {
	people := ascending(6)
	kept, left, guest, admin, gone, demoted := people[0], people[1], people[2], people[3], people[4], people[5]
	roles := map[uuid.UUID]shared.Role{kept: shared.RoleMember, guest: shared.RoleGuest, admin: shared.RoleAdmin, demoted: shared.RoleGuest}
	m := domain.Module{Name: "Auth", Status: domain.ModulePlanned, StartDate: day(t, "2026-11-02"), TargetDate: day(t, "2026-11-13"),
		LeadID: &left, MemberIDs: []uuid.UUID{kept, left, demoted}}
	for _, tt := range []struct {
		name string
		p    domain.ModulePatch
		want []string
	}{
		{"the start alone after the target", domain.ModulePatch{SetStartDate: true, StartDate: day(t, "2026-11-14")}, []string{"start_date not_allowed"}},
		{"the target alone before the start", domain.ModulePatch{SetTargetDate: true, TargetDate: day(t, "2026-11-01")}, []string{"target_date not_allowed"}},
		{"both reversed", domain.ModulePatch{SetStartDate: true, StartDate: day(t, "2026-11-20"), SetTargetDate: true, TargetDate: day(t, "2026-11-19")},
			[]string{"target_date not_allowed"}},
		{"a guest lead", domain.ModulePatch{SetLeadID: true, LeadID: &guest}, []string{"lead_id not_allowed"}},
		{"a lead who is no member", domain.ModulePatch{SetLeadID: true, LeadID: &gone}, []string{"lead_id not_allowed"}},
		{"a kept member, a guest now, made the lead", domain.ModulePatch{SetLeadID: true, LeadID: &demoted}, []string{"lead_id not_allowed"}},
		{"members added who may not be", domain.ModulePatch{MemberIDs: &[]uuid.UUID{guest, kept, gone, admin}},
			[]string{"member_ids[0] not_allowed", "member_ids[2] not_allowed"}},
		{"every problem", domain.ModulePatch{SetTargetDate: true, TargetDate: day(t, "2026-11-01"), SetLeadID: true, LeadID: &guest,
			MemberIDs: &[]uuid.UUID{gone}}, []string{"target_date not_allowed", "lead_id not_allowed", "member_ids[0] not_allowed"}},
	} {
		got, changed, err := tt.p.Apply(m, roles)
		if fields := fieldsOf(t, err); !reflect.DeepEqual(fields, tt.want) || changed || !reflect.DeepEqual(got, domain.Module{}) {
			t.Errorf("%s: %q, %t, %+v; want %q", tt.name, fields, changed, got, tt.want)
		}
	}
	// The lead who has left, given as he is, and the members kept — one who has left, one a guest now — are not looked
	// at; nor is a member removed.
	for _, p := range []domain.ModulePatch{
		{SetLeadID: true, LeadID: &left, MemberIDs: &[]uuid.UUID{demoted, left, kept}},
		{MemberIDs: &[]uuid.UUID{left, admin}},
		{SetLeadID: true, LeadID: &admin, MemberIDs: &[]uuid.UUID{demoted}},
	} {
		if _, _, err := p.Apply(m, roles); err != nil {
			t.Errorf("%+v refused: %v", p, err)
		}
	}
}

// DiffMembers: how a module's members become a change's — the accounts added and those removed, each in id order,
// nil for none; a member both have is in neither, its row left as it is (M6 P2 spec R5).
func TestDiffMembers(t *testing.T) {
	people := ascending(4)
	a, b, c, d := people[0], people[1], people[2], people[3]
	for _, tt := range []struct {
		name     string
		from, to []uuid.UUID
		want     domain.MemberDiff
	}{
		{"the same set in another order", []uuid.UUID{a, b, c}, []uuid.UUID{c, a, b}, domain.MemberDiff{}},
		{"none to none", nil, []uuid.UUID{}, domain.MemberDiff{}},
		{"added and removed", []uuid.UUID{a, b, c}, []uuid.UUID{d, b, a}, domain.MemberDiff{Added: []uuid.UUID{d}, Removed: []uuid.UUID{c}}},
		{"all anew", nil, []uuid.UUID{c, a}, domain.MemberDiff{Added: []uuid.UUID{a, c}}},
		{"all removed", []uuid.UUID{b, d}, []uuid.UUID{}, domain.MemberDiff{Removed: []uuid.UUID{b, d}}},
		{"all replaced", []uuid.UUID{a, b}, []uuid.UUID{d, c}, domain.MemberDiff{Added: []uuid.UUID{c, d}, Removed: []uuid.UUID{a, b}}},
	} {
		if got := domain.DiffMembers(tt.from, tt.to); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v; want %+v", tt.name, got, tt.want)
		}
	}
}
