package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's errors (M3 design 5.3).
var (
	// ErrWorkspaceNotFound answers a workspace that does not exist, is
	// deleted, or of which the caller is not an active member: the
	// workspace module's code, which the project's operations under a
	// workspace declare too (M3 design 5.1).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
	// ErrIdentifierTaken answers an identifier an undeleted project of the
	// workspace has.
	ErrIdentifierTaken = shared.NewError(shared.KindConflict, "project.identifier_taken", "A project of the workspace has this identifier.")
	// ErrNameTaken answers a name an undeleted project of the workspace has.
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
	// ErrNotFound answers a project that does not exist, is deleted, or that
	// the caller does not see (M3 design 3.4, 8.2).
	ErrNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrArchived answers a change of an archived project (M3 design 3.19).
	ErrArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrMemberNotFound answers a project membership that does not exist, is
	// deleted or has ended, or whose project the caller does not see: the
	// same 404 for all (M3 design 5.3, 8.2).
	ErrMemberNotFound = shared.NewError(shared.KindNotFound, "project.member_not_found",
		"The project member does not exist, or you cannot see the project.")
	// ErrOwnMembership answers a removal of the caller's own project
	// membership, and a change of his own project role by one who is not the
	// workspace's admin (M3 design 3.5, 5.3). It names no remedy: leaving,
	// which ends one's own membership, refuses a project's only admin too.
	ErrOwnMembership = shared.NewError(shared.KindConflict, "project.own_membership",
		"You cannot remove your own membership of the project, nor change your own role in it unless you are a workspace admin.")
	// ErrRoleTooHigh answers M3 design 3.5's relative rule: a change of the
	// role of a member whose role is not below the caller's, or to a role
	// not below his, by one who is not the workspace's admin; a removal of a
	// member whose role is above the caller's, by anyone.
	ErrRoleTooHigh = shared.NewError(shared.KindForbidden, "project.role_too_high",
		"The role is too high for you: unless you are a workspace admin, you change only a member whose project role is below yours, to a role "+
			"below yours; and you remove only a member whose project role is not above yours.")
	// ErrSoleAdmin answers an ending of an account's project memberships
	// that would leave a project with other active members without an
	// active admin: he is its only one (M3 design 3.7 rule 2). The
	// workspace's removal and leaving declare it too (M3 design 5.1).
	ErrSoleAdmin = shared.NewError(shared.KindConflict, "project.sole_admin",
		"Ending the membership would leave a project that has other members without an admin; make another of its members an admin first.")
)

// LeadNotAllowed is the 422 of a lead who is not an active admin or member
// of the workspace (M3 design 3.19).
func LeadNotAllowed() error {
	return shared.Invalid(shared.FieldError{Field: "project_lead_id", Code: shared.FieldNotAllowed,
		Message: "must be an active admin or member of the workspace"})
}

// Unassignable is the problem of field, the lead or the default assignee
// of an update, naming an account that is not an active member of the
// project, or is its guest (M3 design 3.19).
func Unassignable(field string) shared.FieldError {
	return shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "must be an active member of the project who is not its guest"}
}
