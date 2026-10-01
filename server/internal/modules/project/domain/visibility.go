package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// Visibility is which of a workspace's projects an active member sees
// besides those he is an active member of (M3 design 3.4): every one, or
// the public ones, or none. It is the access module's rule of seeing a
// project, which a list applies in its query; bootstrap's test holds the
// list equal to reading each project (M3 design 9.3).
type Visibility struct {
	All    bool
	Public bool
}

// VisibilityOf is the Visibility of an active member of workspace role
// role: the workspace's admin sees every project, its member the public
// ones too, its guest no other; by set, as access decides it.
func VisibilityOf(role shared.Role) Visibility {
	switch role {
	case shared.RoleAdmin:
		return Visibility{All: true, Public: true}
	case shared.RoleMember:
		return Visibility{Public: true}
	}
	return Visibility{}
}
