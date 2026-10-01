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
)

// LeadNotAllowed is the 422 of a lead who is not an active admin or member
// of the workspace (M3 design 3.19).
func LeadNotAllowed() error {
	return shared.Invalid(shared.FieldError{Field: "project_lead_id", Code: shared.FieldNotAllowed,
		Message: "must be an active admin or member of the workspace"})
}
