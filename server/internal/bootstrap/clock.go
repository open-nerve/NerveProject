package bootstrap

import (
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

// newClock is shared by serve and the administrator commands. It constructs
// only business time; limiters, River and waiting deadlines use real time.
func newClock(cfg config.Config) clock.Offset {
	return clock.Offset{Base: clock.System{}, Offset: cfg.Clock.Offset}
}
