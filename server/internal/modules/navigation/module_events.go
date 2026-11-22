package navigation

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Subscribers composes the module's transactional consumers independently of
// New: the deletion of a deleted project's views, favorites and recent
// visits, and of a deleted workspace's own views and favorites (P3b spec
// 2.7, P3c spec 2.6, P3d spec 2.6); the cascades on the favorites of a cycle
// or a module deleted or archived, and of a project archived (P3c spec 2.6);
// and the forgetting of every account's recent visits of a deleted work
// item, each of a deleted subtree's (P3d spec 2.6, R10): the last two
// through ports' store in the publishing transaction. The module publishes
// no event.
func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter)), app.NewFavoriteCascades(ports.store),
		app.NewForgetDeletedIssueVisits(ports.store)}
}
