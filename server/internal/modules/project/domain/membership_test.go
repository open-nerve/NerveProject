package domain

import (
	"errors"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The three roles are accepted; any other, between them or outside them,
// is a 422 naming role.
func TestCheckMemberRole(t *testing.T) {
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin} {
		if err := CheckMemberRole(role); err != nil {
			t.Errorf("CheckMemberRole(%d) = %v, want nil", role, err)
		}
	}
	want := []shared.FieldError{{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}}
	for _, role := range []shared.Role{0, 10, 25} {
		var e *shared.Error
		if err := CheckMemberRole(role); !errors.As(err, &e) || e.Code != "validation_failed" || !reflect.DeepEqual(e.Fields, want) {
			t.Errorf("CheckMemberRole(%d) = %v, want validation_failed with %+v", role, err, want)
		}
	}
}

// The callers of the relative rule (M3 design 3.5): their workspace and
// project roles.
var (
	projectAdmin   = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	projectMember  = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
	bothAdmin      = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	adminAsMember  = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleMember, ProjectAdmin: true}
	adminAsGuest   = shared.Grant{WorkspaceRole: shared.RoleAdmin, ProjectRole: shared.RoleGuest, ProjectAdmin: true}
	unknownProject = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: 25}
)

// guestOnly is the 422 of a workspace guest given a role other than a
// guest's.
var guestOnly = shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed, Message: "must be 5: the member is a guest of the workspace"})

// sameProblem reports whether err is want: the same *shared.Error, or a 422
// with the same fields; nil for nil.
func sameProblem(err, want error) bool {
	var e, w *shared.Error
	switch {
	case want == nil || err == nil:
		return err == want
	case !errors.As(err, &e) || !errors.As(want, &w):
		return false
	case w.Code == "validation_failed":
		return e.Code == w.Code && reflect.DeepEqual(e.Fields, w.Fields)
	}
	return e == w
}

// Each half of M3 design 3.5's rule for a change of role, each with the
// case that holds it and its counterexample. One who is not the
// workspace's admin: his own role, never (even unchanged, even to a role
// below his); another's, only from a role below his own to a role below
// his own, so a project admin changes no admin and makes none, and a
// project member changes a guest to a guest at most. The workspace's
// admin, whatever his project role: his own, an admin's, to admin. For
// everyone: a workspace guest is a guest. Below is by roleOrder: a project
// role outside the three has nothing below it, and a role between them
// (10, below 20 by the numbers) is below nothing it is not in roleOrder
// before.
func TestCheckRoleChange(t *testing.T) {
	member, guest, admin := shared.RoleMember, shared.RoleGuest, shared.RoleAdmin
	for _, tt := range []struct {
		name string
		c    RoleChange
		want error
	}{
		{"a project admin makes a member a guest", RoleChange{projectAdmin, false, member, member, guest}, nil},
		{"a project admin makes a guest a member", RoleChange{projectAdmin, false, guest, member, member}, nil},
		{"a project admin keeps a member a member", RoleChange{projectAdmin, false, member, member, member}, nil},
		{"a project admin, his own role, to a member's", RoleChange{projectAdmin, true, admin, member, member}, ErrOwnMembership},
		{"a project admin, his own role, kept", RoleChange{projectAdmin, true, admin, member, admin}, ErrOwnMembership},
		{"a project admin changes another admin", RoleChange{projectAdmin, false, admin, member, member}, ErrRoleTooHigh},
		{"a project admin makes a member an admin", RoleChange{projectAdmin, false, member, member, admin}, ErrRoleTooHigh},
		{"a project admin makes a guest an admin", RoleChange{projectAdmin, false, guest, member, admin}, ErrRoleTooHigh},
		{"a project member keeps a guest a guest", RoleChange{projectMember, false, guest, member, guest}, nil},
		{"a project member makes a member a guest", RoleChange{projectMember, false, member, member, guest}, ErrRoleTooHigh},
		{"a project member makes a guest a member", RoleChange{projectMember, false, guest, member, member}, ErrRoleTooHigh},
		{"a project member, his own role", RoleChange{projectMember, true, member, member, guest}, ErrOwnMembership},
		{"a project role outside the three", RoleChange{unknownProject, false, guest, member, guest}, ErrRoleTooHigh},
		{"a project admin, from a role between the three", RoleChange{projectAdmin, false, 10, member, guest}, ErrRoleTooHigh},
		{"a project admin, to a role between the three", RoleChange{projectAdmin, false, guest, member, 10}, ErrRoleTooHigh},
		{"a workspace admin, his own role, to an admin's", RoleChange{adminAsMember, true, member, admin, admin}, nil},
		{"a workspace admin and project admin, his own role, to a guest's", RoleChange{bothAdmin, true, admin, admin, guest}, nil},
		{"a workspace admin makes the admin a member", RoleChange{adminAsMember, false, admin, member, member}, nil},
		{"a workspace admin makes a member an admin", RoleChange{adminAsMember, false, member, member, admin}, nil},
		{"a workspace admin who is a project guest changes an admin", RoleChange{adminAsGuest, false, admin, member, guest}, nil},
		{"a workspace admin makes another workspace admin a member", RoleChange{bothAdmin, false, admin, admin, member}, nil},
		{"a project admin makes a workspace guest a member", RoleChange{projectAdmin, false, guest, guest, member}, guestOnly},
		{"a workspace admin makes a workspace guest a member", RoleChange{adminAsMember, false, guest, guest, member}, guestOnly},
		{"a workspace admin makes a workspace guest an admin", RoleChange{bothAdmin, false, guest, guest, admin}, guestOnly},
		{"a workspace admin keeps a workspace guest a guest", RoleChange{bothAdmin, false, guest, guest, guest}, nil},
	} {
		if err := CheckRoleChange(tt.c); !sameProblem(err, tt.want) {
			t.Errorf("%s: CheckRoleChange(%+v) = %v, want %v", tt.name, tt.c, err, tt.want)
		}
	}
}

// M3 design 3.5's rule for a removal: nobody his own membership, whatever
// his roles; nobody a member whose project role is above his own, the
// workspace's admins neither; an equal or lower role, anyone the decision
// lets remove. Above is by roleOrder: a project role outside the three has
// nothing up to it, and a role between them is up to nothing.
func TestCheckRemoval(t *testing.T) {
	for _, tt := range []struct {
		name   string
		caller shared.Role
		own    bool
		role   shared.Role
		want   error
	}{
		{"an admin removes an admin", shared.RoleAdmin, false, shared.RoleAdmin, nil},
		{"an admin removes a member", shared.RoleAdmin, false, shared.RoleMember, nil},
		{"an admin removes a guest", shared.RoleAdmin, false, shared.RoleGuest, nil},
		{"an admin, his own", shared.RoleAdmin, true, shared.RoleAdmin, ErrOwnMembership},
		{"a member, his own", shared.RoleMember, true, shared.RoleMember, ErrOwnMembership},
		{"a member removes a member", shared.RoleMember, false, shared.RoleMember, nil},
		{"a member removes a guest", shared.RoleMember, false, shared.RoleGuest, nil},
		{"a member removes an admin", shared.RoleMember, false, shared.RoleAdmin, ErrRoleTooHigh},
		{"a guest removes a guest", shared.RoleGuest, false, shared.RoleGuest, nil},
		{"a guest removes a member", shared.RoleGuest, false, shared.RoleMember, ErrRoleTooHigh},
		{"a project role outside the three", 25, false, shared.RoleGuest, ErrRoleTooHigh},
		{"an admin removes a role between the three", shared.RoleAdmin, false, 10, ErrRoleTooHigh},
	} {
		if err := CheckRemoval(tt.caller, tt.own, tt.role); !sameProblem(err, tt.want) {
			t.Errorf("%s: CheckRemoval(%d, %v, %d) = %v, want %v", tt.name, tt.caller, tt.own, tt.role, err, tt.want)
		}
	}
}
