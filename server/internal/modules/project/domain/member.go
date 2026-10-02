package domain

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Member is an active membership of a project, as the members' list shows
// it (M3 design 5.2): its id, the project, the member's account, his role
// in the project and when he became its member.
type Member struct {
	ID        uuid.UUID
	ProjectID uuid.UUID
	MemberID  uuid.UUID
	Role      shared.Role
	CreatedAt time.Time
}

// NewMember is an account to add to a project, with the role he is to
// have in it.
type NewMember struct {
	MemberID uuid.UUID
	Role     shared.Role
}

// MaxNewMembers is the most members one request adds (M3 design 5.1).
const MaxNewMembers = 100

// projectRoles are the three roles a member of a project can have.
var projectRoles = []shared.Role{shared.RoleAdmin, shared.RoleMember, shared.RoleGuest}

// CheckNewMembers checks what the request alone tells: 1 to MaxNewMembers
// members, each role one of the three, each account named once. Every
// problem is reported at once, as one 422 validation_failed; whether each
// account may be added is the use case's, under its locks (M3 design 3.6
// convention 3).
func CheckNewMembers(members []NewMember) error {
	var found []*shared.FieldError
	switch {
	case len(members) == 0:
		found = append(found, &shared.FieldError{Field: "members", Code: shared.FieldTooShort, Message: "must name a member"})
	case len(members) > MaxNewMembers:
		found = append(found, &shared.FieldError{Field: "members", Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must name at most %d members", MaxNewMembers)})
	}
	for i, m := range members {
		if slices.ContainsFunc(members[:i], func(o NewMember) bool { return o.MemberID == m.MemberID }) {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldDuplicate,
				Message: "is listed before"})
		}
		if !slices.Contains(projectRoles, m.Role) {
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].role", i), Code: shared.FieldInvalidFormat,
				Message: "is not 5, 15 or 20"})
		}
	}
	return invalid(found...)
}

// addable are the project roles an active member of the workspace can be
// added to a project with, by his workspace role (M3 design 3.5; Plane
// views/project/member.py:69-83): a workspace admin as an admin alone, a
// workspace guest as a guest alone, a workspace member as any of the three.
var addable = map[shared.Role][]shared.Role{
	shared.RoleAdmin:  {shared.RoleAdmin},
	shared.RoleMember: projectRoles,
	shared.RoleGuest:  {shared.RoleGuest},
}

// CanAdd reports whether an active member of the workspace of workspace
// role workspaceRole may be added to a project as role. The sets are
// named, not bounds.
func CanAdd(workspaceRole, role shared.Role) bool {
	return slices.Contains(addable[workspaceRole], role)
}

// Target is an account a request adds to a project, as the use case reads
// him under its locks: his role in the workspace, nil while he is not its
// active member, and whether he is an active member of the project.
type Target struct {
	NewMember
	WorkspaceRole *shared.Role
	Member        bool
}

// CheckTargets checks each account to add against what the use case read
// under its locks, after the decision (M3 design 3.5, 3.6 convention 3):
// an active member of the workspace, not an active member of the project
// already, added with a role his workspace role allows (CanAdd). One 422
// names each one refused, by his place in the request.
func CheckTargets(targets []Target) error {
	var found []*shared.FieldError
	for i, t := range targets {
		switch {
		case t.WorkspaceRole == nil:
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldNotAllowed,
				Message: "must be an active member of the workspace"})
		case t.Member:
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].member_id", i), Code: shared.FieldDuplicate,
				Message: "is an active member of the project already"})
		case !CanAdd(*t.WorkspaceRole, t.Role):
			found = append(found, &shared.FieldError{Field: fmt.Sprintf("members[%d].role", i), Code: shared.FieldNotAllowed,
				Message: "is not one his workspace role allows: a workspace admin joins as an admin, a guest as a guest"})
		}
	}
	return invalid(found...)
}

// roleOrder is the project roles from the least to the most.
var roleOrder = []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin}

// JoinRole is the project role of an account who joins a project, of
// workspace role workspaceRole (M3 design 3.5, 3.6 convention 6): his
// workspace role for a new membership; for his ended one, ended, the
// lesser of its role and his workspace role, so that a restored membership
// gives no more than a new one would, nor more than it had. The order is
// roleOrder's, not the numbers'.
func JoinRole(ended *shared.Role, workspaceRole shared.Role) shared.Role {
	if ended == nil || slices.Index(roleOrder, workspaceRole) < slices.Index(roleOrder, *ended) {
		return workspaceRole
	}
	return *ended
}
