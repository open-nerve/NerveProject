package notification

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Subscribers composes the module's transactional consumers independently of
// New: the deletion of a deleted work item's notifications, and of a deleted
// project's (M7 P1b spec 2.8). The module publishes no event.
func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewForgetDeletedIssue(ports.store), app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter))}
}
