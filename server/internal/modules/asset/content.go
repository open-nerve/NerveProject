package asset

import (
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ContentHost = app.ContentHost
type ContentBindRequest = app.ContentBindRequest
type ContentBinding = app.ContentBinding
type DraftAssetRequest = app.DraftAssetRequest
type DraftAssetTransfer = app.DraftAssetTransfer

// Owner modules supply already authorized, locked parents. These ports operate
// only on asset rows within the caller's transaction.
func NewContentBindings(p Provided, clock app.Clock) ContentBinding {
	return app.NewContentBindings(app.ContentBindingDeps{Locks: p.store, Writer: p.store, Clock: clock})
}
func NewDraftAssetTransfer(p Provided, clock app.Clock, events shared.EventPublisher) DraftAssetTransfer {
	return app.NewDraftAssetTransfer(app.DraftAssetDeps{Locks: p.store, Reader: p.store, Writer: p.store, Clock: clock, Events: events})
}
