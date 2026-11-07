package app

import (
	"context"
	"uuid"
)

// ProjectFacts are a project's facts as the planning module reads them (M6 P1
// spec 2.1): its workspace, whether it is archived, its time zone, which a
// cycle's status is read in, and its cycle and module toggles.
type ProjectFacts struct {
	WorkspaceID           uuid.UUID
	Archived              bool
	Timezone              string
	CycleView, ModuleView bool
}

// Projects is what the module asks of a project: its facts without a lock,
// for a read and a write's checks before its lock, and the same facts under
// the project's FOR NO KEY UPDATE, the first lock of a write (M6 design 6).
// found is false for a project that is not there or is deleted, also one
// deleted while the lock waited. project.Provide's PlanningProjects is it,
// converted (bootstrap).
type Projects interface {
	Facts(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	Lock(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
}
