package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// deletionEvent is the event kind of the deletion of entity, a cycle or a
// module of the workspace's project, which by deleted at at (M6 P3a spec
// 2.5, R6): its one change its name before the deletion, which the issue
// module, which cannot read planning's tables, writes into the activity of
// the work items it takes out of entity.
func deletionEvent(kind string, entity shared.EntityRef, workspace, project uuid.UUID, name string, at time.Time, by uuid.UUID) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, OccurredAt: at, WorkspaceID: workspace, ProjectID: project, ActorID: &by, Entity: entity,
		Changes: []shared.Change{{Field: "name", Old: &name}}}
}
