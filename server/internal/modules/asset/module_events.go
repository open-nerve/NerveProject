package asset

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return app.DeletionSubscribers(ports.store, riveradapter.NewScopeDeletions(inserter))
}
