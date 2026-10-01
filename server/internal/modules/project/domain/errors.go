package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The project module's errors (M3 design 5.3).
var (
	// ErrIdentifierTaken answers an identifier an undeleted project of the
	// workspace has.
	ErrIdentifierTaken = shared.NewError(shared.KindConflict, "project.identifier_taken", "A project of the workspace has this identifier.")
	// ErrNameTaken answers a name an undeleted project of the workspace has.
	ErrNameTaken = shared.NewError(shared.KindConflict, "project.name_taken", "A project of the workspace has this name.")
)
