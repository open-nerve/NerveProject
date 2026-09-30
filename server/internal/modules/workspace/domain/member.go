package domain

import (
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Membership is a row of workspace_members: an account's membership of a
// workspace, active or ended (M3 design 5.2).
type Membership struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	IsActive    bool
	CreatedAt   time.Time
}

// MemberUser is a member's public profile as another member sees it (M3
// design 5.2): Email is nil for a caller whose role may not see it.
type MemberUser struct {
	ID          uuid.UUID
	DisplayName string
	FirstName   string
	LastName    string
	Email       *string
}

// Member is a membership with its member's profile: WorkspaceMember.
type Member struct {
	Membership
	User MemberUser
}

// roles are the three workspace roles (Plane's ROLE_CHOICES).
var roles = []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin}

// CheckMemberRole checks that role is one of the three, as one 422
// validation_failed.
func CheckMemberRole(role shared.Role) error {
	if !slices.Contains(roles, role) {
		return shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"})
	}
	return nil
}

// emailReaders are the workspace roles that see the members' addresses:
// admins and members, not guests (Plane views/workspace/member.py:50-54,
// M3 design 3.4). Roles are compared by set, never by order.
var emailReaders = []shared.Role{shared.RoleAdmin, shared.RoleMember}

// SeesEmails reports whether a caller of role sees the members' addresses.
func SeesEmails(role shared.Role) bool {
	return slices.Contains(emailReaders, role)
}
