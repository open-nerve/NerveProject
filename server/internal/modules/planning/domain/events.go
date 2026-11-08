package domain

// The events the module publishes, each in the transaction of the deletion
// it tells of (M6 P3a spec 2.5, R6): a cycle's and a module's. Each names
// the deleted cycle or module, its workspace and project, the account that
// deleted it and the moment, and, as its one change, its name before the
// deletion, which a module that cannot read the cycles' and modules' tables
// writes into the work items' activity.
const (
	EventCycleDeleted  = "planning.cycle_deleted"
	EventModuleDeleted = "planning.module_deleted"
)
