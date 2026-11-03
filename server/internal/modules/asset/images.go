package asset

import (
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
)

type ImageBindingHost = app.ImageBindingHost
type ImageChange = app.ImageChange
type ImageBindingRequest = app.ImageBindingRequest
type ImageBindings = app.ImageBindings

// NewImageBindings needs only asset storage and the business clock, so owner
// modules can receive a complete binding port before asset's owner access exists.
func NewImageBindings(p Provided, clock app.Clock) ImageBindings {
	return app.NewImageBindings(app.ImageBindingDeps{Locks: p.store, Writer: p.store, Bindings: p.store, Clock: clock})
}

type ImageReference = app.ImageReference
type ImageURL = app.ImageURL
type ImageReader = app.ImageReader

func NewImageReader(p Provided, signer app.ReadSigner, runtimeNow func() time.Time) ImageReader {
	return app.NewImageReader(app.ImageReadDeps{Locks: p.store, Signer: signer, RuntimeNow: runtimeNow})
}
