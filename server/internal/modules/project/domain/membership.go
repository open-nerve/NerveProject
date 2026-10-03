package domain

import (
	"slices"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CheckMemberRole checks the role a change of a project membership gives,
// from the request alone: one of the three, else a 422 naming role.
func CheckMemberRole(role shared.Role) error {
	if !slices.Contains(roleOrder, role) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"})
	}
	return nil
}

// assignable are the project roles a change of a membership can give its
// member, by his workspace role (M3 design 3.5; Plane
// views/project/member.py:257-261): a workspace guest a guest's alone, a
// workspace member or admin any of the three. The sets are named, not
// bounds.
var assignable = map[shared.Role][]shared.Role{
	shared.RoleAdmin:  roleOrder,
	shared.RoleMember: roleOrder,
	shared.RoleGuest:  {shared.RoleGuest},
}

// RoleChange is a change of a project membership's role as the use case
// reads it under its locks: the caller's grant, whether the membership is
// his own, its member's role in the project and in the workspace, and the
// role it is to have.
type RoleChange struct {
	Caller        shared.Grant
	Own           bool
	From          shared.Role
	WorkspaceRole shared.Role
	To            shared.Role
}

// CheckRoleChange checks c against M3 design 3.5, after the decision let
// the caller change roles (the project's admins, and its members who are
// the workspace's admins). One who is not the workspace's admin changes
// neither his own role (ErrOwnMembership), nor the role of a member whose
// role is not below his own, nor gives a role that is not below his own
// (ErrRoleTooHigh): a project admin promotes nobody to admin and changes no
// other admin. Then, for every caller, a workspace guest's role stays a
// guest's (422 role not_allowed). Below is roleOrder's, never the
// numbers': a role outside the three has no role below it and is below
// none.
func CheckRoleChange(c RoleChange) error {
	if c.Caller.WorkspaceRole != shared.RoleAdmin {
		below := rolesBelow(c.Caller.ProjectRole)
		switch {
		case c.Own:
			return ErrOwnMembership
		case !slices.Contains(below, c.From), !slices.Contains(below, c.To):
			return ErrRoleTooHigh
		}
	}
	if !slices.Contains(assignable[c.WorkspaceRole], c.To) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldNotAllowed,
			Message: "must be 5: the member is a guest of the workspace"})
	}
	return nil
}

// CheckRemoval checks a removal of a project membership of role, the
// caller's own when own is set, after the decision let the caller remove
// members (M3 design 3.5): nobody removes his own membership
// (ErrOwnMembership: he leaves the project), nor one whose role is above
// his own project role, callerRole (ErrRoleTooHigh), the workspace's admins
// neither (Plane views/project/member.py:290-321 gives them no exception).
func CheckRemoval(callerRole shared.Role, own bool, role shared.Role) error {
	switch {
	case own:
		return ErrOwnMembership
	case !slices.Contains(rolesUpTo(callerRole), role):
		return ErrRoleTooHigh
	}
	return nil
}

// rolesBelow are the roles below role in roleOrder: none for a role outside
// it.
func rolesBelow(role shared.Role) []shared.Role {
	i := slices.Index(roleOrder, role)
	if i < 0 {
		return nil
	}
	return slices.Clone(roleOrder[:i])
}

// rolesUpTo are role and the roles below it in roleOrder: none for a role
// outside it.
func rolesUpTo(role shared.Role) []shared.Role {
	i := slices.Index(roleOrder, role)
	if i < 0 {
		return nil
	}
	return slices.Clone(roleOrder[:i+1])
}
