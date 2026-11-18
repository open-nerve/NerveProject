package app

import (
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// memberRule accepts the rule of an action only when it allows the active
// members of the target project alone, alike on every work item of it: a
// project-level rule without AllowCreator (M7 P1c spec 2.5). A LevelVisible
// rule, which allows the workspace's admin and, on a public project, its
// member, though neither is a member of it; a workspace-level one; one that
// allows the creator; and an action without a rule are errors. Each rule of
// the table it accepts decides so: it refuses every caller who is no active
// member of the project, and whether he created the work item changes none
// of its decisions.
func TestMemberRuleAcceptsOnlyAMembersRule(t *testing.T) {
	for _, tt := range []struct {
		action shared.Action
		ok     bool
	}{
		{"issue.read", true}, {"issue.update", true}, {"project.read", false}, {"project.join", false}, {"workspace.read", false},
		{"issue.delete", false}, {"cycle.delete", false}, {"no.such.action", false},
	} {
		if _, err := memberRule(tt.action); (err == nil) != tt.ok {
			t.Errorf("memberRule(%s) = %v; want accepted %v", tt.action, err, tt.ok)
		}
	}
	roles := []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest, 10}
	members := []domain.Membership{{}}
	for _, r := range roles {
		members = append(members, domain.Membership{Active: true, Role: r})
	}
	projects := []*domain.Project{nil} // none: the target's project is not there
	for _, public := range []bool{false, true} {
		for _, all := range []bool{false, true} {
			for _, m := range members {
				projects = append(projects, &domain.Project{Public: public, Member: m, GuestViewAll: all})
			}
		}
	}
	accepted := 0
	for _, action := range domain.RuleKeys() {
		rule, err := memberRule(action)
		if err != nil {
			continue
		}
		accepted++
		for _, ws := range members {
			for _, p := range projects {
				f := domain.Facts{Workspace: ws, Project: p}
				grant, err := domain.Decide(rule, f)
				f.Creator = true
				creatorGrant, creatorErr := domain.Decide(rule, f)
				if grant != creatorGrant || (err == nil) != (creatorErr == nil) ||
					errors.Is(err, shared.ErrNotVisible) != errors.Is(creatorErr, shared.ErrNotVisible) {
					t.Errorf("%s on %+v, %+v: %+v, %v, and for its creator %+v, %v", action, ws, p, grant, err, creatorGrant, creatorErr)
				}
				if err == nil && (p == nil || !p.Member.Active) {
					t.Errorf("%s on %+v, %+v allows one who is no member of the project", action, ws, p)
				}
			}
		}
	}
	if accepted == 0 {
		t.Error("memberRule accepts no rule of the table")
	}
}
