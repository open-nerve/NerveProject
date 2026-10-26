package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LinkCrawls queues metadata work in the caller's transaction.
type LinkCrawls interface {
	Enqueue(context.Context, uuid.UUID, string) error
}

// NewCrawlOnLinkChange consumes the URL change without rereading the link.
func NewCrawlOnLinkChange(crawls LinkCrawls) shared.EventSubscriber {
	return shared.EventSubscriber{
		Name:  "crawl_on_link_change",
		Types: []string{domain.EventLinkCreated, domain.EventLinkUpdated},
		Handle: func(ctx context.Context, event shared.Event) error {
			var url *string
			for _, change := range event.Changes {
				if change.Field != "url" {
					continue
				}
				if url != nil || change.New == nil {
					return errors.New("link change event requires one non-nil URL change")
				}
				url = change.New
			}
			if url == nil {
				if event.Type == domain.EventLinkCreated {
					return errors.New("link created event requires a URL change")
				}
				return nil
			}
			return crawls.Enqueue(ctx, event.Entity.ID, *url)
		},
	}
}
