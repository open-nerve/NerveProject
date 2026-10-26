package app_test

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// discardEvents explicitly supplies a publisher to tests of existing writes.
// Event semantics and failures are exercised by focused publishing tests.
type discardEvents struct{}

func (discardEvents) Publish(context.Context, shared.Event) error { return nil }
