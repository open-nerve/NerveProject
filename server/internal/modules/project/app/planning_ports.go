package app

import "uuid"

// PlanningFacts are what project.Provide hands the planning module about a
// project (M6 P1 spec 2.1): its workspace, whether it is archived, its time
// zone, which a cycle's status is read in, and its cycle and module toggles.
// bootstrap converts them into planning's value.
type PlanningFacts struct {
	WorkspaceID           uuid.UUID
	Archived              bool
	Timezone              string
	CycleView, ModuleView bool
}
