package project

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// NavigationTargets are the projects a page of favorites names, the
// navigation module's targets and their projects (M7 P3c spec 2.4, 2.5),
// read by project ids, in one statement, without a lock, in the
// transaction ctx carries or on the pool. project.Provide builds them over
// the pool; bootstrap converts them into navigation's ProjectTargets.
type NavigationTargets interface {
	// NavigationTargets reads the undeleted projects among ids, archived or
	// not, of any workspace, by project; a deleted project and an id of
	// none are not among them.
	NavigationTargets(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]NavigationTarget, error)
}

// NavigationTarget is the value NavigationTargets hands over: bootstrap
// converts it into navigation's.
type NavigationTarget = app.NavigationTarget
