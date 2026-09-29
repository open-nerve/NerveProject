package shared_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The roles are the values of Plane's ROLE_CHOICES, which the tables store
// (M3 design 3.4).
func TestRolesArePlanes(t *testing.T) {
	if shared.RoleGuest != 5 || shared.RoleMember != 15 || shared.RoleAdmin != 20 {
		t.Errorf("roles = %d, %d, %d; want 5, 15, 20", shared.RoleGuest, shared.RoleMember, shared.RoleAdmin)
	}
}

// A use case recognizes ErrNotVisible through wrapping, and no module's 404
// is mistaken for it: those have a code.
func TestErrNotVisible(t *testing.T) {
	wrapped := fmt.Errorf("authorize: %w", shared.ErrNotVisible)
	if !errors.Is(wrapped, shared.ErrNotVisible) {
		t.Errorf("errors.Is(%v, ErrNotVisible) = false, want true", wrapped)
	}
	notFound := shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist.")
	for _, err := range []error{notFound, shared.Forbidden(), errors.New("not visible")} {
		if errors.Is(err, shared.ErrNotVisible) {
			t.Errorf("errors.Is(%v, ErrNotVisible) = true, want false", err)
		}
	}
	if got := shared.ErrNotVisible.ProblemStatus(); got != 404 || shared.ErrNotVisible.ProblemCode() != "" {
		t.Errorf("ErrNotVisible = %d %q, want 404 without a code", got, shared.ErrNotVisible.ProblemCode())
	}
}
