package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// LinkMetadataWriter updates metadata only while the link still matches the task.
type LinkMetadataWriter interface {
	SetLinkMetadata(context.Context, uuid.UUID, string, domain.LinkMetadata) (bool, error)
}

// CrawlLinkDeps are the crawl and guarded persistence ports.
type CrawlLinkDeps struct {
	Crawler LinkCrawler
	Links   LinkMetadataWriter
}

// CrawlLink fetches metadata and writes it without request-side locks or audit.
type CrawlLink struct {
	crawler LinkCrawler
	links   LinkMetadataWriter
}

func NewCrawlLink(d CrawlLinkDeps) *CrawlLink {
	return &CrawlLink{crawler: d.Crawler, links: d.Links}
}

// Execute treats stale tasks and stored network failures as completed work.
// Persistence errors retain their chain so the worker can retry them.
func (c *CrawlLink) Execute(ctx context.Context, id uuid.UUID, url string) error {
	metadata := c.crawler.Crawl(ctx, url)
	_, err := c.links.SetLinkMetadata(ctx, id, url, metadata)
	return err
}
