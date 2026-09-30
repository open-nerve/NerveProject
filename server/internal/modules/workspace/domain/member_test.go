package domain

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Admins and members see the addresses; guests, and a role outside the
// three, do not.
func TestSeesEmails(t *testing.T) {
	for role, want := range map[shared.Role]bool{
		shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false, 25: false,
	} {
		if got := SeesEmails(role); got != want {
			t.Errorf("SeesEmails(%d) = %v, want %v", role, got, want)
		}
	}
}
