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

// Projects is what the module asks of a project: its facts without a lock
// (FactsReader), and the same facts under the project's FOR NO KEY UPDATE,
// the first lock of a write (M6 design 6); found is false for a project
// that is not there or is deleted, also one deleted while the lock waited.
// And its members' roles (RoleReader). project.Provide's PlanningProjects
// is it, converted (bootstrap).
type Projects interface {
	FactsReader
	Lock(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	RoleReader
}

// FactsReader reads a project's facts without a lock, in the transaction
// ctx carries, if any: for a read, a write's checks before its lock, and
// the issue targets, whose asking transaction holds the project's lock
// (IssueTargets). found is false for a project that is not there or is
// deleted.
type FactsReader interface {
	Facts(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
}

// RoleReader reads the project roles of those of the accounts ids that are
// the project's active members, by account; one who is not, never or no
// longer, has none. A module's lead and members are held to them under the
// project's lock (M6 P2 spec R5), which keeps them as read.
type RoleReader interface {
	Roles(ctx context.Context, project uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error)
}
