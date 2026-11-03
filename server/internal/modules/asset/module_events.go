package asset

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return app.DeletionSubscribers(ports.store, riveradapter.NewScopeDeletions(inserter))
}

// EventTypes declares the transactional attachment events owned by asset.
func EventTypes() []string {
	return []string{domain.EventAttachmentAdded, domain.EventAttachmentDeleted}
}
