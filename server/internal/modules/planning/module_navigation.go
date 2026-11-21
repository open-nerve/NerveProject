package planning

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
)

// NavigationTargets are the cycles and the modules a page of favorites
// names (M7 P3c spec 2.4, 2.5), read by their ids, in one statement each,
// without a lock, in the transaction ctx carries or on the pool, whatever
// their projects' toggles. planning.Provide builds them over the pool;
// bootstrap converts them into navigation's PlanningTargets.
type NavigationTargets interface {
	// NavigationCycles reads the undeleted cycles among ids, archived or
	// not, of any workspace, by cycle; a deleted one and an id of none are
	// not among them.
	NavigationCycles(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationTarget, error)
	// NavigationModules reads the undeleted modules among ids as
	// NavigationCycles reads the cycles.
	NavigationModules(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationTarget, error)
}

// NavigationTarget is the value NavigationTargets hands over: bootstrap
// converts it into navigation's.
type NavigationTarget = app.NavigationTarget
