package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
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
// (FactsReader); the same facts under the project's FOR NO KEY UPDATE, the
// first lock of a write (M6 design 6); and under its FOR SHARE, the lock of
// a change of one's own display settings on a page (M6 P3b spec R6), which
// waits for a write's and a deletion's lock and they for it. found is false
// for a project that is not there or is deleted, also one deleted while the
// lock waited. And its members' roles (RoleReader), and its states' groups
// (StateGroupReader). project.Provide's PlanningProjects is it, converted
// (bootstrap).
type Projects interface {
	FactsReader
	Lock(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	Share(ctx context.Context, id uuid.UUID) (f ProjectFacts, found bool, err error)
	RoleReader
	StateGroupReader
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

// StateGroupReader reads the group of each of a project's undeleted states,
// its triage state too, by state, in the transaction ctx carries (M6 P4
// spec R2): the issue module answers a work item's state by its id, and a
// progress counts the work item by its state's group. Under the project's
// lock, which every write of a state takes too, they stay as read.
type StateGroupReader interface {
	StateGroups(ctx context.Context, projectID uuid.UUID) (map[uuid.UUID]domain.StateGroup, error)
}
