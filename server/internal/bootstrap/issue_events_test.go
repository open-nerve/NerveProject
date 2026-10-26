package bootstrap

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// discardIssueEvents keeps existing write fixtures explicit about their
// publisher; publishing integration tests supply their own dispatcher.
type discardIssueEvents struct{}

func (discardIssueEvents) Publish(context.Context, shared.Event) error { return nil }
