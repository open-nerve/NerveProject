// Package app holds the instance module's use cases and the ports they need.
package app

import (
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

// Clock tells the time. platform/clock implements it.
type Clock interface {
	Now() time.Time
}

// InfoSource reports the build this instance runs. GetInfo declares it;
// adapter/buildinfo implements it.
type InfoSource interface {
	Build() domain.Build
}
