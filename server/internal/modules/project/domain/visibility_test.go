package domain

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each workspace role sees what access lets it see (M3 design 3.4); a role
// outside the three sees nothing more than its own projects.
func TestVisibilityOf(t *testing.T) {
	for role, want := range map[shared.Role]Visibility{shared.RoleAdmin: {All: true, Public: true}, shared.RoleMember: {Public: true},
		shared.RoleGuest: {}, 10: {}, 25: {}} {
		if got := VisibilityOf(role); got != want {
			t.Errorf("VisibilityOf(%d) = %+v, want %+v", role, got, want)
		}
	}
}
