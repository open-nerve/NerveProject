package navigation

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Subscribers composes the module's transactional consumers independently of
// New: the deletion of the views of a deleted project, and of a deleted
// workspace's own (P3b spec 2.7). The module publishes no event.
func Subscribers(inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter))}
}
