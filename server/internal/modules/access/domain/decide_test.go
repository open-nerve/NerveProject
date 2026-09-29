package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// outcome is a cell of a decision table: allowed, 403 forbidden or 404 not
// visible.
type outcome string

const (
	allowed   outcome = "ok"
	forbidden outcome = "403"
	invisible outcome = "404"
)

// outcomeOf names what Decide answered; any other error fails the test.
func outcomeOf(t *testing.T, err error) outcome {
	t.Helper()
	switch {
	case err == nil:
		return allowed
	case errors.Is(err, shared.ErrNotVisible):
		return invisible
	case errors.Is(err, shared.Forbidden()):
		return forbidden
	}
	t.Fatalf("Decide() = %v, want nil, ErrNotVisible or Forbidden", err)
	return ""
}

var (
	admin  = domain.Membership{Active: true, Role: shared.RoleAdmin}
	member = domain.Membership{Active: true, Role: shared.RoleMember}
	guest  = domain.Membership{Active: true, Role: shared.RoleGuest}
	none   = domain.Membership{}
)

// The identities of M3 design 9.2's workspace-level columns, as the
// WorkspaceRoles port reports them. The three that are not active members
// keep the role of their row: only Active may count.
var workspaceIdentities = []struct {
	name string
	ws   domain.Membership
}{
	{"admin", admin},
	{"member", member},
	{"guest", guest},
	{"never a member", none},
	{"removed", domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"workspace deleted", domain.Membership{Active: false, Role: shared.RoleMember}},
	{"role outside the three", domain.Membership{Active: true, Role: 10}},
}

func TestDecideAtTheWorkspaceLevel(t *testing.T) {
	all := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}
	tests := []struct {
		name  string
		roles []shared.Role
		want  []outcome // in the order of workspaceIdentities
	}{
		{"every role", all, []outcome{allowed, allowed, allowed, invisible, invisible, invisible, forbidden}},
		{"admins and members", all[:2], []outcome{allowed, allowed, forbidden, invisible, invisible, invisible, forbidden}},
		{"admins", all[:1], []outcome{allowed, forbidden, forbidden, invisible, invisible, invisible, forbidden}},
		{"no role", nil, []outcome{forbidden, forbidden, forbidden, invisible, invisible, invisible, forbidden}},
	}
	for _, tt := range tests {
		rule := domain.Rule{Level: domain.LevelWorkspace, Roles: tt.roles}
		for i, id := range workspaceIdentities {
			grant, err := domain.Decide(rule, domain.Facts{Workspace: id.ws})
			if got := outcomeOf(t, err); got != tt.want[i] {
				t.Errorf("%s, %s: Decide() = %s, want %s", tt.name, id.name, got, tt.want[i])
			}
			if err == nil && grant != (shared.Grant{WorkspaceRole: id.ws.Role}) {
				t.Errorf("%s, %s: Grant = %+v, want the workspace role %d alone", tt.name, id.name, grant, id.ws.Role)
			}
		}
	}
}

// A workspace-level rule reads no project: facts about one change nothing,
// neither the answer nor the Grant, whether the project is one the caller
// could see or not.
func TestTheWorkspaceLevelIgnoresTheProject(t *testing.T) {
	admins := domain.Rule{Level: domain.LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}}
	p := &domain.Project{Public: true, Member: admin}
	if _, err := domain.Decide(admins, domain.Facts{Workspace: member, Project: p}); outcomeOf(t, err) != forbidden {
		t.Errorf("a member who administers a project: %v, want forbidden", err)
	}
	every := domain.Rule{Level: domain.LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}}
	for _, tt := range []struct {
		name string
		p    *domain.Project
	}{
		{"a private project he is not a member of", &domain.Project{}},
		{"a public project he administers", &domain.Project{Public: true, Member: admin}},
	} {
		grant, err := domain.Decide(every, domain.Facts{Workspace: member, Project: tt.p})
		if err != nil || grant != (shared.Grant{WorkspaceRole: shared.RoleMember}) {
			t.Errorf("a member, %s: Decide() = %+v, %v; want the workspace role 15 alone", tt.name, grant, err)
		}
	}
}

// The columns of M3 design 9.2's project-level table, as the ports report
// them: PA, PM and PG are project admin, member and guest (workspace member,
// member, guest); WM demoted to PG a workspace member whose project role is
// guest (3.5), who counts as PG: the project role decides; PM+WA a project
// member who is the workspace's admin; WA-, WM- and WG- a workspace admin,
// member and guest who are not project members; P-before a workspace member
// who left a private project where he was its admin, and P-before public
// the same on a public project, who counts as WM- public (9.2's note); X a
// caller who is not an active workspace member, though his project
// membership is still active. A membership that is not active keeps the
// role of its row, as in workspaceIdentities: only Active may count.
var projectIdentities = []struct {
	name   string
	ws     domain.Membership
	public bool
	member domain.Membership
}{
	{"PA", member, false, admin},
	{"PM", member, false, member},
	{"PG", guest, false, guest},
	{"WM demoted to PG", member, false, guest},
	{"PM+WA", admin, false, member},
	{"WA- private", admin, false, none},
	{"WM- public", member, true, none},
	{"WM- private", member, false, none},
	{"WG- public", guest, true, none},
	{"WG- private", guest, false, none},
	{"P-before", member, false, domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"P-before public", member, true, domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"X", domain.Membership{Active: false, Role: shared.RoleMember}, true, admin},
	// 10 lies between the roles and 25 above them: seeing is by set, never
	// by order.
	{"workspace role outside the three, public", domain.Membership{Active: true, Role: 10}, true, none},
	{"workspace role above the three, public", domain.Membership{Active: true, Role: 25}, true, none},
	{"project role outside the three", member, false, domain.Membership{Active: true, Role: 10}},
	{"workspace role outside the three, project admin", domain.Membership{Active: true, Role: 10}, false, admin},
}

func TestDecideAtTheProjectLevels(t *testing.T) {
	all := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}
	const ok, no, hidden = allowed, forbidden, invisible
	tests := []struct {
		name string
		rule domain.Rule
		want []outcome // in the order of projectIdentities
	}{
		// updateProject, addProjectMembers, createState… (9.2)
		{"project admins", domain.Rule{Level: domain.LevelProject, Roles: all[:1]},
			[]outcome{ok, no, no, no, ok, no, no, hidden, hidden, hidden, hidden, no, hidden, hidden, hidden, no, no}},
		{"project admins and members", domain.Rule{Level: domain.LevelProject, Roles: all[:2]},
			[]outcome{ok, ok, no, no, ok, no, no, hidden, hidden, hidden, hidden, no, hidden, hidden, hidden, no, no}},
		// listStates, listLabels, getProjectPreferences… (9.2)
		{"every project role", domain.Rule{Level: domain.LevelProject, Roles: all},
			[]outcome{ok, ok, ok, ok, ok, no, no, hidden, hidden, hidden, hidden, no, hidden, hidden, hidden, no, no}},
		// getProject (9.2)
		{"seeing the project", domain.Rule{Level: domain.LevelVisible},
			[]outcome{ok, ok, ok, ok, ok, ok, ok, hidden, hidden, hidden, hidden, ok, hidden, hidden, hidden, no, no}},
	}
	for _, tt := range tests {
		for i, id := range projectIdentities {
			facts := domain.Facts{Workspace: id.ws, Project: &domain.Project{Public: id.public, Member: id.member}}
			_, err := domain.Decide(tt.rule, facts)
			if got := outcomeOf(t, err); got != tt.want[i] {
				t.Errorf("%s, %s: Decide() = %s, want %s", tt.name, id.name, got, tt.want[i])
			}
			// A project that does not count (none, deleted, another
			// workspace's) is seen by no one.
			facts.Project = nil
			if _, err := domain.Decide(tt.rule, facts); outcomeOf(t, err) != invisible {
				t.Errorf("%s, %s, no project: Decide() = %v, want not visible", tt.name, id.name, err)
			}
		}
	}
}

// A role outside the three is allowed nothing, whatever its value, at every
// level and wherever the decision reads a role: roles are a set, never an
// order. At the workspace level such a workspace role is refused (403), as
// the tables' role outside the three; the workspace level reads no project
// role. At the project levels, what the caller sees picks 404 or 403: a
// caller who is not a member of the project sees it only as the workspace's
// admin or, when it is public, as a workspace member, so a role outside the
// three sees no project it is not a member of, public or private (404, as
// the unknown-role public identities); a caller who is an active member of
// the project sees it and is refused (403, as the unknown-role project admin
// and project member), the workspace's admin too: being the workspace's
// admin exempts no role from the three.
func TestRolesOutsideTheThreeAreAllowedNothing(t *testing.T) {
	// Every gap between the three, their edges and the smallint's ends.
	unknown := []shared.Role{-32768, -1, 0, 1, 4, 6, 10, 14, 16, 17, 19, 21, 25, 100, 32767}
	all := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}
	type namedRule struct {
		name string
		rule domain.Rule
	}
	// Every row of the table, and a rule of each level with each set of
	// roles the tables above use: the table has no project-level row yet.
	var rules []namedRule
	for _, action := range domain.RuleKeys() {
		rule, _ := domain.RuleFor(action)
		rules = append(rules, namedRule{string(action), rule})
	}
	for _, level := range []domain.Level{domain.LevelWorkspace, domain.LevelProject} {
		for _, roles := range [][]shared.Role{all, all[:2], all[:1], nil} {
			rules = append(rules, namedRule{fmt.Sprintf("level %d, roles %v", level, roles), domain.Rule{Level: level, Roles: roles}})
		}
	}
	rules = append(rules, namedRule{"seeing the project", domain.Rule{Level: domain.LevelVisible}})

	active := func(r shared.Role) domain.Membership { return domain.Membership{Active: true, Role: r} }
	positions := []struct {
		name        string
		facts       func(r shared.Role, public bool) domain.Facts
		atWorkspace outcome // "" when the workspace level reads no role there
		atProject   outcome
	}{
		{"the workspace role of a caller not in the project", func(r shared.Role, public bool) domain.Facts {
			return domain.Facts{Workspace: active(r), Project: &domain.Project{Public: public}}
		}, forbidden, invisible},
		{"the workspace role of the project's admin", func(r shared.Role, public bool) domain.Facts {
			return domain.Facts{Workspace: active(r), Project: &domain.Project{Public: public, Member: admin}}
		}, forbidden, forbidden},
		{"the project role of a workspace member", func(r shared.Role, public bool) domain.Facts {
			return domain.Facts{Workspace: member, Project: &domain.Project{Public: public, Member: active(r)}}
		}, "", forbidden},
		{"the project role of the workspace's admin", func(r shared.Role, public bool) domain.Facts {
			return domain.Facts{Workspace: admin, Project: &domain.Project{Public: public, Member: active(r)}}
		}, "", forbidden},
	}
	for _, nr := range rules {
		for _, pos := range positions {
			want := pos.atProject
			if nr.rule.Level == domain.LevelWorkspace {
				want = pos.atWorkspace
			}
			if want == "" {
				continue
			}
			for _, r := range unknown {
				for _, public := range []bool{false, true} {
					_, err := domain.Decide(nr.rule, pos.facts(r, public))
					if got := outcomeOf(t, err); got != want {
						t.Errorf("%s, %s = %d, public %t: Decide() = %s, want %s", nr.name, pos.name, r, public, got, want)
					}
				}
			}
		}
	}
}

// The Grant carries the roles the decision read, and ProjectAdmin for a
// project member who is its admin or the workspace's; a membership that is
// not active gives no project role. It is the same Grant at every level that
// allows the caller: LevelVisible, and for a project member LevelProject,
// by his project role or as the workspace's admin.
func TestTheGrantCarriesTheRoles(t *testing.T) {
	visible := domain.Rule{Level: domain.LevelVisible}
	everyRole := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}}
	admins := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}}
	tests := []struct {
		name  string
		ws    domain.Membership
		p     domain.Project
		rules []domain.Rule // each allows the caller
		want  shared.Grant
	}{
		{"PA", member, domain.Project{Member: admin}, []domain.Rule{visible, everyRole, admins},
			shared.Grant{WorkspaceRole: 15, ProjectRole: 20, ProjectAdmin: true}},
		{"PM", member, domain.Project{Member: member}, []domain.Rule{visible, everyRole},
			shared.Grant{WorkspaceRole: 15, ProjectRole: 15}},
		{"PG", guest, domain.Project{Member: guest}, []domain.Rule{visible, everyRole},
			shared.Grant{WorkspaceRole: 5, ProjectRole: 5}},
		// admins allows him as the workspace's admin, not by his project role.
		{"PM+WA", admin, domain.Project{Member: member}, []domain.Rule{visible, everyRole, admins},
			shared.Grant{WorkspaceRole: 20, ProjectRole: 15, ProjectAdmin: true}},
		{"WA-", admin, domain.Project{}, []domain.Rule{visible},
			shared.Grant{WorkspaceRole: 20}},
		{"WM- public", member, domain.Project{Public: true}, []domain.Rule{visible},
			shared.Grant{WorkspaceRole: 15}},
		{"P-before public", member, domain.Project{Public: true, Member: domain.Membership{Active: false, Role: shared.RoleAdmin}}, []domain.Rule{visible},
			shared.Grant{WorkspaceRole: 15}},
	}
	for _, tt := range tests {
		for _, rule := range tt.rules {
			p := tt.p
			grant, err := domain.Decide(rule, domain.Facts{Workspace: tt.ws, Project: &p})
			if err != nil || grant != tt.want {
				t.Errorf("%s, level %d, roles %v: Decide() = %+v, %v; want %+v", tt.name, rule.Level, rule.Roles, grant, err, tt.want)
			}
		}
	}
}

// A rule of no known level allows nothing, and is not a refusal the caller
// could act on: an internal error.
func TestARuleOfNoKnownLevelIsAnError(t *testing.T) {
	for _, level := range []domain.Level{0, 4} {
		rule := domain.Rule{Level: level, Roles: []shared.Role{shared.RoleAdmin}}
		_, err := domain.Decide(rule, domain.Facts{Workspace: admin, Project: &domain.Project{Member: admin}})
		var se *shared.Error
		if err == nil || errors.As(err, &se) {
			t.Errorf("level %d: Decide() = %v, want an internal error", level, err)
		}
	}
}
