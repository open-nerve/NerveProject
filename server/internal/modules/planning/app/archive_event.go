package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// archiveEvent is the event kind of the archive of entity, a cycle or a
// module of the workspace's project, which by archived at at, the moment
// written into its archived_at (M7 P3c spec 2.6, R3): it carries no change.
func archiveEvent(kind string, entity shared.EntityRef, workspace, project uuid.UUID, at time.Time, by uuid.UUID) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: at, WorkspaceID: workspace, ProjectID: project, ActorID: &by, Entity: entity}
}
