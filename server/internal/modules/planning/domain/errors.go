package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The planning module's errors (M6 P1 spec 2.3). Its operations on a project
// answer the project module's codes for the project, which they declare too.
var (
	// ErrProjectNotFound answers a project that does not exist, is deleted,
	// or that the caller does not see (M3 design 3.4, 8.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrProjectArchived answers a write on the cycles of an archived
	// project (M6 design 3.1).
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrCycleNotFound answers a cycle that does not exist or is deleted, and
	// one whose project the caller does not see, all with this one value.
	ErrCycleNotFound = shared.NewError(shared.KindNotFound, "planning.cycle_not_found", "The cycle does not exist, or you cannot see it.")
	// ErrCyclesDisabled answers a write on the cycles of a project whose
	// cycles are turned off (M6 design 3.8).
	ErrCyclesDisabled = shared.NewError(shared.KindConflict, "planning.cycles_disabled", "Cycles are turned off in this project; turn them on to change them.")
	// ErrArchived answers a change of an archived cycle (M6 design 3.1).
	ErrArchived = shared.NewError(shared.KindConflict, "planning.archived", "The cycle is archived; unarchive it to change it.")
	// ErrCycleCompleted answers a change of a completed cycle other than its
	// order (M6 design 3.1).
	ErrCycleCompleted = shared.NewError(shared.KindConflict, "planning.cycle_completed", "The cycle is completed; only its order can change.")
	// ErrCycleNotCompleted answers the archiving of a cycle that is not
	// completed (M6 design 3.6).
	ErrCycleNotCompleted = shared.NewError(shared.KindConflict, "planning.cycle_not_completed", "Only a completed cycle can be archived.")
	// ErrDatesOverlap answers dates that share a day with another cycle of
	// the project (M6 design 3.1).
	ErrDatesOverlap = shared.NewError(shared.KindConflict, "planning.cycle_dates_overlap", "The dates overlap another cycle of this project.")
)
