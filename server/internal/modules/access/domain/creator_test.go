package domain_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AllowCreator lets a project-level rule allow the work item's creator too,
// while he is an active member of the project of the role member or admin
// (M4 design 3.4): not as its guest, the project's flag for guests
// whichever, not once his membership has ended, and not with a project role
// outside the three. A caller who did not create it is decided by his role
// alone, and a rule without AllowCreator does not look at the creator.
func TestAllowCreatorAllowsAMemberWhoCreatedIt(t *testing.T) {
	admins := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}}
	byCreator := domain.Rule{Level: domain.LevelProject, Roles: []shared.Role{shared.RoleAdmin}, AllowCreator: true}
	left := domain.Membership{Active: false, Role: shared.RoleMember}
	tests := []struct {
		name    string
		rule    domain.Rule
		ws      domain.Membership
		p       domain.Project
		creator bool
		want    outcome
	}{
		{"a member who created it", byCreator, member, domain.Project{Member: member}, true, allowed},
		{"a member who did not", byCreator, member, domain.Project{Member: member}, false, forbidden},
		{"a member who created it, by a rule without AllowCreator", admins, member, domain.Project{Member: member}, true, forbidden},
		{"the project's admin who did not create it", byCreator, member, domain.Project{Member: admin}, false, allowed},
		{"the project's admin who created it, by a rule of its creator alone", domain.Rule{Level: domain.LevelProject, AllowCreator: true},
			member, domain.Project{Member: admin}, true, allowed},
		{"a guest who created it", byCreator, guest, domain.Project{Member: guest}, true, forbidden},
		{"a guest who created it, the project showing its guests every work item", byCreator, guest,
			domain.Project{Member: guest, GuestViewAll: true}, true, forbidden},
		{"a member demoted to the project's guest who created it", byCreator, member, domain.Project{Member: guest}, true, forbidden},
		{"a creator who left the private project", byCreator, member, domain.Project{Member: left}, true, invisible},
		{"a creator who left the public project", byCreator, member, domain.Project{Public: true, Member: left}, true, forbidden},
		{"a creator whose project role is outside the three", byCreator, member,
			domain.Project{Member: domain.Membership{Active: true, Role: 10}}, true, forbidden},
	}
	for _, tt := range tests {
		p := tt.p
		_, err := domain.Decide(tt.rule, domain.Facts{Workspace: tt.ws, Project: &p, Creator: tt.creator})
		if got := outcomeOf(t, err); got != tt.want {
			t.Errorf("%s: Decide() = %s, want %s", tt.name, got, tt.want)
		}
	}
}
