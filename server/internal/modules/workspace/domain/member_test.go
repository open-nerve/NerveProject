package domain

import (
	"errors"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The three roles pass; any other value is one 422 on role, whatever its
// size (roles are a set, not a scale).
func TestCheckMemberRole(t *testing.T) {
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin} {
		if err := CheckMemberRole(role); err != nil {
			t.Errorf("CheckMemberRole(%d) = %v, want nil", role, err)
		}
	}
	want := []shared.FieldError{{Field: "role", Code: "invalid_format", Message: "is not 5, 15 or 20"}}
	for _, role := range []shared.Role{0, 4, 10, 16, 21, 25, -5} {
		err := CheckMemberRole(role)
		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || !slices.Equal(se.Fields, want) {
			t.Errorf("CheckMemberRole(%d) = %#v, want validation_failed with %v", role, err, want)
		}
	}
}

// Admins and members see the addresses; guests, and a role outside the
// three, do not.
func TestSeesEmails(t *testing.T) {
	for role, want := range map[shared.Role]bool{
		shared.RoleAdmin: true, shared.RoleMember: true, shared.RoleGuest: false, 0: false, 10: false, 16: false, 25: false,
	} {
		if got := SeesEmails(role); got != want {
			t.Errorf("SeesEmails(%d) = %v, want %v", role, got, want)
		}
	}
}
