package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's errors (M4 design 5.3). Its operations on a project
// answer the project module's codes for the project, which they declare
// too (M4 design 5.1), with the project module's details.
var (
	// ErrProjectNotFound answers a project that does not exist, is deleted,
	// or that the caller does not see (M3 design 3.4, 8.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrProjectArchived answers a write on the work items of an archived
	// project (M4 design 3.8).
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
)
