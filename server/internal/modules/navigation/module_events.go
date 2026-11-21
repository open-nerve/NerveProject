package navigation

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Subscribers composes the module's transactional consumers independently of
// New: the deletion of the views and the favorites of a deleted project, and
// of a deleted workspace's own (P3b spec 2.7, P3c spec 2.6); and the
// cascades on the favorites of a cycle or a module deleted or archived, and
// of a project archived, through ports' store in the publishing transaction
// (P3c spec 2.6). The module publishes no event.
func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter)), app.NewFavoriteCascades(ports.store)}
}
