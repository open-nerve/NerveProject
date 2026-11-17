package project

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// NotificationNames are the names of a work item's project and state the
// notification module's fan-out completes a notification's snapshot with
// (M7 P1b spec 2.5, 2.6), each read alone, by its ids, and without a lock:
// the fan-out holds a work item's row and takes no lock of a project, which
// the writes take before a work item's. Each reads in the transaction ctx
// carries. project.Provide builds them over the pool; bootstrap converts
// them into notification's ProjectNames.
type NotificationNames interface {
	// NotificationState reads the name and the group of the project's
	// undeleted state stateID, its triage state too; found is false when
	// there is none.
	NotificationState(ctx context.Context, projectID, stateID uuid.UUID) (s NotificationState, found bool, err error)
	// NotificationIdentifier reads the identifier of the workspace's
	// undeleted project projectID, archived or not; found is false when
	// there is none.
	NotificationIdentifier(ctx context.Context, workspaceID, projectID uuid.UUID) (identifier string, found bool, err error)
}

// NotificationState is the value NotificationNames hands over: bootstrap
// converts it into notification's.
type NotificationState = app.NotificationState
