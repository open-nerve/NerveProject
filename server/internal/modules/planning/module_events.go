package planning

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// EventTypes declares the event types the module publishes: a cycle's and a
// module's deletion (M6 P3a spec R6), and their archive (M7 P3c spec 2.6).
func EventTypes() []string {
	return []string{domain.EventCycleDeleted, domain.EventModuleDeleted, domain.EventCycleArchived, domain.EventModuleArchived}
}

// Subscribers composes the module's transactional consumers independently of
// New: the deletion of a deleted project's cycles.
func Subscribers(inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter))}
}
