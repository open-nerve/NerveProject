package domain_test

import (
	"maps"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// tableCells is every row of the rule table decided for every identity of
// M3 design 9.2, the matrix's cells without HTTP: a row deleted, widened or
// narrowed fails here, and so does a row added to the table without its
// cells here. A workspace-level row has a cell per workspaceIdentities, a
// project-level one per projectIdentities.
var tableCells = map[shared.Action][]outcome{
	// admin, member, guest, never a member, removed, workspace deleted, a role outside the three
	"workspace.read":               {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace.update":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace.delete":             {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace.leave":              {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_member.list":        {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_member.update":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_member.remove":      {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_preferences.read":   {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_preferences.update": {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"workspace_invitation.list":    {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.create":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.update":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"workspace_invitation.delete":  {allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden},
	"project.list":                 {allowed, allowed, allowed, invisible, invisible, invisible, forbidden},
	"project.create":               {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
	"project_identifier.check":     {allowed, allowed, forbidden, invisible, invisible, invisible, forbidden},
	// PA, PM, PG, WM demoted to PG, PM+WA, WA- private, WM- public, WM- private, WG- public, WG- private, P-before,
	// P-before public, X, workspace role outside the three public, above the three public, project role outside the
	// three, workspace role outside the three project admin (projectIdentities)
	"project.read": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
	"project.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.archive": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.unarchive": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project.delete": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_preferences.read": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_preferences.update": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_member.list": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"project_member.add": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project.read: the guest's 403 is the use case's (M3 design 3.5).
	"project.join": {allowed, allowed, allowed, allowed, allowed, allowed, allowed, invisible, invisible, invisible, invisible, allowed,
		invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.add: the relative rules are the use case's (M3
	// design 3.5).
	"project_member.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.update.
	"project_member.remove": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible,
		invisible, forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project_member.list: every active member of the project.
	"project.leave": {allowed, allowed, allowed, allowed, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	// As project.update: not the project's guests.
	"state.create": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
	"state.update": {allowed, forbidden, forbidden, forbidden, allowed, forbidden, forbidden, invisible, invisible, invisible, invisible,
		forbidden, invisible, invisible, invisible, forbidden, forbidden},
}

// cells decides rule for each identity of its level.
func cells(t *testing.T, rule domain.Rule) []outcome {
	t.Helper()
	var out []outcome
	if rule.Level == domain.LevelWorkspace {
		for _, id := range workspaceIdentities {
			_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws})
			out = append(out, outcomeOf(t, err))
		}
		return out
	}
	for _, id := range projectIdentities {
		_, err := domain.Decide(rule, domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member}})
		out = append(out, outcomeOf(t, err))
	}
	return out
}

func TestEveryRuleDecidesItsCells(t *testing.T) {
	if got, want := domain.RuleKeys(), slices.Sorted(maps.Keys(tableCells)); !slices.Equal(got, want) {
		t.Fatalf("the rule table has rows %q; the cells here are for %q", got, want)
	}
	for _, action := range domain.RuleKeys() {
		rule, ok := domain.RuleFor(action)
		if !ok {
			t.Fatalf("RuleFor(%q) found no row, though RuleKeys lists it", action)
		}
		if got := cells(t, rule); !slices.Equal(got, tableCells[action]) {
			t.Errorf("%s decides %q, want %q", action, got, tableCells[action])
		}
	}
}

// Asking whether an identifier is free is for whoever may create a project
// with it (spec §3 item 13): the two rows are one rule, the roles as a set.
func TestTheIdentifierCheckIsCreatesRule(t *testing.T) {
	check, okCheck := domain.RuleFor("project_identifier.check")
	create, okCreate := domain.RuleFor("project.create")
	if !okCheck || !okCreate || check.Level != create.Level ||
		!slices.Equal(slices.Sorted(slices.Values(check.Roles)), slices.Sorted(slices.Values(create.Roles))) {
		t.Errorf("project_identifier.check = %+v, %v; want project.create's %+v, %v", check, okCheck, create, okCreate)
	}
}

// RuleFor hands out a copy: a caller that writes to the row's roles changes
// no later decision.
func TestRuleForReturnsACopy(t *testing.T) {
	for _, action := range domain.RuleKeys() {
		rule, _ := domain.RuleFor(action)
		want := slices.Clone(rule.Roles)
		for i := range rule.Roles {
			rule.Roles[i] = 10
		}
		if again, _ := domain.RuleFor(action); !slices.Equal(again.Roles, want) {
			t.Errorf("RuleFor(%q) after a caller's write: roles %v, want %v", action, again.Roles, want)
		}
	}
}

// An action the table has no row for is found by no lookup: the Authorizer
// refuses it.
func TestAnActionWithoutARowHasNoRule(t *testing.T) {
	for _, action := range []shared.Action{"", "no.such.action", "WORKSPACE.READ", "workspace.read "} {
		if rule, ok := domain.RuleFor(action); ok {
			t.Errorf("RuleFor(%q) = %+v, want none", action, rule)
		}
	}
}
