package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
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
// deleted while the lock waited. Roles reads the project roles of those of
// the accounts ids that are its active members, by account: a module's lead
// and members are held to them under the lock (M6 P2 spec R5), which keeps
// them as read. project.Provide's PlanningProjects is it, converted
// (bootstrap).
type Projects interface {
	Facts(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	Lock(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	Roles(ctx context.Context, project uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}
