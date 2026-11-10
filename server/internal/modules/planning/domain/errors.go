package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The planning module's errors (M6 P1 spec 2.3, P2 spec 2.3). Its operations
// on a project answer the project module's codes for the project, which they
// declare too.
var (
	// ErrProjectNotFound answers a project that does not exist, is deleted,
	// or that the caller does not see (M3 design 3.4, 8.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrProjectArchived answers a write on the cycles or the modules of an
	// archived project (M6 design 3.1, 3.2).
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrCycleNotFound answers a cycle that does not exist or is deleted, and
	// one whose project the caller does not see, all with this one value.
	ErrCycleNotFound = shared.NewError(shared.KindNotFound, "planning.cycle_not_found", "The cycle does not exist, or you cannot see it.")
	// ErrCyclesDisabled answers a write on the cycles of a project whose
	// cycles are turned off (M6 design 3.8).
	ErrCyclesDisabled = shared.NewError(shared.KindConflict, "planning.cycles_disabled", "Cycles are turned off in this project; turn them on to change them.")
	// ErrArchived answers a change of an archived cycle or module, and a
	// write on the links of an archived module (M6 design 3.6, P2 spec R13).
	ErrArchived = shared.NewError(shared.KindConflict, "planning.archived", "The cycle or module is archived; unarchive it to change it.")
	// ErrCycleCompleted answers a change of a completed cycle other than its
	// order (M6 design 3.1).
	ErrCycleCompleted = shared.NewError(shared.KindConflict, "planning.cycle_completed", "The cycle is completed; only its order can change.")
	// ErrCycleNotCompleted answers the archiving of a cycle that is not
	// completed (M6 design 3.6).
	ErrCycleNotCompleted = shared.NewError(shared.KindConflict, "planning.cycle_not_completed", "Only a completed cycle can be archived.")
	// ErrTransferFromOpenCycle answers a transfer of the work items of a
	// cycle not completed on the write's day in its project's time zone (M6
	// P4 spec 2.3, R7): ErrCycleNotCompleted's code, a transfer's message.
	ErrTransferFromOpenCycle = shared.NewError(shared.KindConflict, "planning.cycle_not_completed",
		"Only a completed cycle can transfer its work items.")
	// ErrTransferToCompletedCycle answers a transfer to a cycle completed on
	// the write's day in its project's time zone (M6 P4 spec 2.3, R7):
	// ErrCycleCompleted's code, a transfer's message.
	ErrTransferToCompletedCycle = shared.NewError(shared.KindConflict, "planning.cycle_completed", "A completed cycle cannot take work items.")
	// ErrDatesOverlap answers dates that share a day with another cycle of
	// the project (M6 design 3.1).
	ErrDatesOverlap = shared.NewError(shared.KindConflict, "planning.cycle_dates_overlap", "The dates overlap another cycle of this project.")
	// ErrModuleNotFound answers a module that does not exist or is deleted,
	// and one whose project the caller does not see, all with this one value
	// (M6 P2 spec 2.3).
	ErrModuleNotFound = shared.NewError(shared.KindNotFound, "planning.module_not_found", "The module does not exist, or you cannot see it.")
	// ErrModulesDisabled answers a write on the modules of a project whose
	// modules are turned off (M6 design 3.8).
	ErrModulesDisabled = shared.NewError(shared.KindConflict, "planning.modules_disabled", "Modules are turned off in this project; turn them on to change them.")
	// ErrModuleNameTaken answers a name an undeleted module of the project
	// has, archived or not, in any case (M6 P2 spec R8).
	ErrModuleNameTaken = shared.NewError(shared.KindConflict, "planning.module_name_taken", "A module of this project has this name.")
	// ErrModuleNotClosed answers the archiving of a module that is neither
	// completed nor cancelled (M6 P2 spec R3).
	ErrModuleNotClosed = shared.NewError(shared.KindConflict, "planning.module_not_closed", "Only a completed or cancelled module can be archived.")
	// ErrModuleLinkNotFound answers a module link that does not exist or is
	// deleted, one whose module is deleted, and one whose project the caller
	// does not see, all with this one value (M6 P2 spec 2.3).
	ErrModuleLinkNotFound = shared.NewError(shared.KindNotFound, "planning.module_link_not_found", "The link does not exist, or you cannot see it.")
	// ErrModuleLinkExists answers an address an undeleted link of the module
	// has (M6 design 3.2).
	ErrModuleLinkExists = shared.NewError(shared.KindConflict, "planning.module_link_exists", "This module already has this link.")
)
