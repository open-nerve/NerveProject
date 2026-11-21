package domain

// The events the module publishes, each in the transaction of the write it
// tells of. A cycle's and a module's deletion (M6 P3a spec 2.5, R6): each
// names the deleted cycle or module, its workspace and project, the account
// that deleted it and the moment, and, as its one change, its name before
// the deletion, which a module that cannot read the cycles' and modules'
// tables writes into the work items' activity. A cycle's and a module's
// archive (M7 P3c spec 2.6, R3): each names the archived cycle or module, its
// workspace and project, the account that archived it and the moment written
// into its archived_at, with no change; an unarchive, and an archive of one
// archived already, which writes nothing, publish none.
const (
	EventCycleDeleted   = "planning.cycle_deleted"
	EventModuleDeleted  = "planning.module_deleted"
	EventCycleArchived  = "planning.cycle_archived"
	EventModuleArchived = "planning.module_archived"
)
