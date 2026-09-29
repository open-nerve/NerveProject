package domain_test

import (
	"errors"
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

// A workspace-level rule reads no project: facts about one change nothing.
func TestTheWorkspaceLevelIgnoresTheProject(t *testing.T) {
	rule := domain.Rule{Level: domain.LevelWorkspace, Roles: []shared.Role{shared.RoleAdmin}}
	p := &domain.Project{Public: true, Member: admin}
	if _, err := domain.Decide(rule, domain.Facts{Workspace: member, Project: p}); outcomeOf(t, err) != forbidden {
		t.Errorf("a member who administers a project: %v, want forbidden", err)
	}
}

// The columns of M3 design 9.2's project-level table, as the ports report
// them: PA, PM and PG are project admin, member and guest (workspace member,
// member, guest); PM+WA a project member who is the workspace's admin; WA-,
// WM- and WG- a workspace admin, member and guest who are not project
// members; P-before a workspace member who left a private project where he
// was its admin; X a caller who is not an active workspace member, though
// his project membership is still active.
var projectIdentities = []struct {
	name   string
	ws     domain.Membership
	public bool
	member domain.Membership
}{
	{"PA", member, false, admin},
	{"PM", member, false, member},
	{"PG", guest, false, guest},
	{"PM+WA", admin, false, member},
	{"WA- private", admin, false, none},
	{"WM- public", member, true, none},
	{"WM- private", member, false, none},
	{"WG- public", guest, true, none},
	{"WG- private", guest, false, none},
	{"P-before", member, false, domain.Membership{Active: false, Role: shared.RoleAdmin}},
	{"X", none, true, admin},
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
			[]outcome{ok, no, no, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, hidden, no, no}},
		{"project admins and members", domain.Rule{Level: domain.LevelProject, Roles: all[:2]},
			[]outcome{ok, ok, no, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, hidden, no, no}},
		// listStates, listLabels, getProjectPreferences… (9.2)
		{"every project role", domain.Rule{Level: domain.LevelProject, Roles: all},
			[]outcome{ok, ok, ok, ok, no, no, hidden, hidden, hidden, hidden, hidden, hidden, hidden, no, no}},
		// getProject (9.2)
		{"seeing the project", domain.Rule{Level: domain.LevelVisible},
			[]outcome{ok, ok, ok, ok, ok, ok, hidden, hidden, hidden, hidden, hidden, hidden, hidden, no, no}},
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

// The Grant carries the roles the decision read, and ProjectAdmin for a
// project member who is its admin or the workspace's.
func TestTheGrantCarriesTheRoles(t *testing.T) {
	visible := domain.Rule{Level: domain.LevelVisible}
	tests := []struct {
		name string
		ws   domain.Membership
		p    domain.Project
		want shared.Grant
	}{
		{"PA", member, domain.Project{Member: admin}, shared.Grant{WorkspaceRole: 15, ProjectRole: 20, ProjectAdmin: true}},
		{"PM", member, domain.Project{Member: member}, shared.Grant{WorkspaceRole: 15, ProjectRole: 15}},
		{"PG", guest, domain.Project{Member: guest}, shared.Grant{WorkspaceRole: 5, ProjectRole: 5}},
		{"PM+WA", admin, domain.Project{Member: member}, shared.Grant{WorkspaceRole: 20, ProjectRole: 15, ProjectAdmin: true}},
		{"WA-", admin, domain.Project{}, shared.Grant{WorkspaceRole: 20}},
		{"WM- public", member, domain.Project{Public: true}, shared.Grant{WorkspaceRole: 15}},
	}
	for _, tt := range tests {
		p := tt.p
		grant, err := domain.Decide(visible, domain.Facts{Workspace: tt.ws, Project: &p})
		if err != nil || grant != tt.want {
			t.Errorf("%s: Decide() = %+v, %v; want %+v", tt.name, grant, err, tt.want)
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
